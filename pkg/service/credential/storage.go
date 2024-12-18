package credential

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/TBD54566975/ssi-sdk/credential"
	"github.com/TBD54566975/ssi-sdk/credential/signing"
	statussdk "github.com/TBD54566975/ssi-sdk/credential/status"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	credint "github.com/tbd54566975/ssi-service/internal/credential"
	"github.com/tbd54566975/ssi-service/internal/keyaccess"
	"github.com/tbd54566975/ssi-service/internal/util"
	"github.com/tbd54566975/ssi-service/pkg/storage"
)

type StoreCredentialRequest struct {
	credint.Container
}

type StoredCredential struct {
	// This ID is generated by the storage module upon first write
	ID string `json:"id"`

	CredentialID string `json:"credentialId"`

	// only one of these fields should be present
	Credential    *credential.VerifiableCredential `json:"credential,omitempty"`
	CredentialJWT *keyaccess.JWT                   `json:"token,omitempty"`

	Issuer       string `json:"issuer"`
	Subject      string `json:"subject"`
	Schema       string `json:"schema"`
	IssuanceDate string `json:"issuanceDate"`
	Revoked      bool   `json:"revoked"`
	Suspended    bool   `json:"suspended"`
}

type WriteContext struct {
	namespace string
	key       string
	value     []byte
}

func (sc StoredCredential) IsValid() bool {
	return sc.ID != "" && (sc.HasDataIntegrityCredential() || sc.HasJWTCredential())
}

func (sc StoredCredential) HasDataIntegrityCredential() bool {
	return sc.Credential != nil && sc.Credential.Proof != nil
}

func (sc StoredCredential) HasJWTCredential() bool {
	return sc.CredentialJWT != nil
}

const (
	credentialNamespace           = "credential"
	statusListCredentialNamespace = "status-list-credential"
	statusListIndexNamespace      = "status-list-index"

	statusListIndexesKey = "status-list-indexes"
	currentListIndexKey  = "current-list-index"

	// A a minimum revocation bitString length of 131,072, or 16KB uncompressed
	bitStringLength = 8 * 1024 * 16

	credentialNotFoundErrMsg = "credential not found"
)

type Storage struct {
	db storage.ServiceStorage
}

type StatusListIndex struct {
	Index int `json:"index"`
}

func NewCredentialStorage(db storage.ServiceStorage) (*Storage, error) {
	if db == nil {
		return nil, util.LoggingNewError("bolt db reference is nil")
	}

	listIndexKeyExists, err := db.Exists(context.Background(), statusListIndexNamespace, currentListIndexKey)
	if err != nil {
		return nil, util.LoggingErrorMsg(err, "exists for currentListIndexKey")
	}

	statusListIndexesKeyExists, err := db.Exists(context.Background(), statusListIndexNamespace, statusListIndexesKey)
	if err != nil {
		return nil, util.LoggingErrorMsg(err, "exists for statusListIndexesKey")
	}

	if listIndexKeyExists != statusListIndexesKeyExists {
		return nil, util.LoggingNewError("list index and status list index not in the same state")
	}

	indexKeysDontExist := !listIndexKeyExists && !statusListIndexesKeyExists

	if indexKeysDontExist {
		randUniqueList := randomUniqueNum(bitStringLength)
		uniqueNumBytes, err := json.Marshal(randUniqueList)
		if err != nil {
			return nil, util.LoggingErrorMsg(err, "could not marshal random unique numbers")
		}

		if err := db.Write(context.Background(), statusListIndexNamespace, statusListIndexesKey, uniqueNumBytes); err != nil {
			return nil, util.LoggingErrorMsg(err, "problem writing status list indexes to db")
		}

		statusListIndexBytes, err := json.Marshal(StatusListIndex{Index: 0})
		if err != nil {
			return nil, util.LoggingErrorMsg(err, "could not marshal status list index bytes")
		}

		if err := db.Write(context.Background(), statusListIndexNamespace, currentListIndexKey, statusListIndexBytes); err != nil {
			return nil, util.LoggingErrorMsg(err, "problem writing current list index to db")
		}
	}

	return &Storage{db: db}, nil
}

func (cs *Storage) GetNextStatusListRandomIndex(ctx context.Context) (int, error) {

	gotUniqueNumBytes, err := cs.db.Read(ctx, statusListIndexNamespace, statusListIndexesKey)
	if err != nil {
		return -1, util.LoggingErrorMsgf(err, "reading status list")
	}

	if len(gotUniqueNumBytes) == 0 {
		return -1, util.LoggingNewErrorf("could not get unique numbers from db")
	}

	var uniqueNums []int
	if err = json.Unmarshal(gotUniqueNumBytes, &uniqueNums); err != nil {
		return -1, util.LoggingErrorMsgf(err, "unmarshalling unique numbers")
	}

	gotCurrentListIndexBytes, err := cs.db.Read(ctx, statusListIndexNamespace, currentListIndexKey)
	if err != nil {
		return -1, util.LoggingErrorMsgf(err, "could not get list index")
	}

	var statusListIndex StatusListIndex
	if err = json.Unmarshal(gotCurrentListIndexBytes, &statusListIndex); err != nil {
		return -1, util.LoggingErrorMsgf(err, "unmarshalling unique numbers")
	}

	return uniqueNums[statusListIndex.Index], nil
}

func (cs *Storage) WriteMany(ctx context.Context, writeContexts []WriteContext) error {
	namespaces := make([]string, 0)
	keys := make([]string, 0)
	values := make([][]byte, 0)

	for i := range writeContexts {
		namespaces = append(namespaces, writeContexts[i].namespace)
		keys = append(keys, writeContexts[i].key)
		values = append(values, writeContexts[i].value)
	}

	return cs.db.WriteMany(ctx, namespaces, keys, values)
}

func (cs *Storage) IncrementStatusListIndexTx(ctx context.Context, tx storage.Tx) error {
	wc, err := cs.GetIncrementStatusListIndexWriteContext(ctx)
	if err != nil {
		return util.LoggingErrorMsg(err, "problem getting increment status listIndex writeContext")
	}

	if err := tx.Write(ctx, wc.namespace, wc.key, wc.value); err != nil {
		return util.LoggingErrorMsg(err, "problem writing current list index to db")
	}

	return nil
}

func (cs *Storage) IncrementStatusListIndex(ctx context.Context) error {
	wc, err := cs.GetIncrementStatusListIndexWriteContext(ctx)
	if err != nil {
		return util.LoggingErrorMsg(err, "problem getting increment status listIndex writeContext")
	}

	if err := cs.db.Write(ctx, wc.namespace, wc.key, wc.value); err != nil {
		return util.LoggingErrorMsg(err, "problem writing current list index to db")
	}

	return nil
}

func (cs *Storage) GetIncrementStatusListIndexWriteContext(ctx context.Context) (*WriteContext, error) {
	gotCurrentListIndexBytes, err := cs.db.Read(ctx, statusListIndexNamespace, currentListIndexKey)
	if err != nil {
		return nil, util.LoggingErrorMsg(err, "could not get list index")
	}

	var statusListIndex StatusListIndex
	if err = json.Unmarshal(gotCurrentListIndexBytes, &statusListIndex); err != nil {
		return nil, util.LoggingErrorMsg(err, "unmarshalling unique numbers")
	}

	if statusListIndex.Index >= bitStringLength-1 {
		return nil, util.LoggingErrorMsg(err, "no more indexes available for status list index")
	}

	statusListIndexBytes, err := json.Marshal(StatusListIndex{Index: statusListIndex.Index + 1})
	if err != nil {
		return nil, util.LoggingErrorMsg(err, "could not marshal status list index bytes")
	}

	wc := WriteContext{
		namespace: statusListIndexNamespace,
		key:       currentListIndexKey,
		value:     statusListIndexBytes,
	}

	return &wc, nil
}

func (cs *Storage) StoreCredential(ctx context.Context, request StoreCredentialRequest) error {
	return cs.storeCredential(ctx, request, credentialNamespace)
}

func (cs *Storage) StoreStatusListCredentialTx(ctx context.Context, tx storage.Tx, request StoreCredentialRequest, statusPurpose statussdk.StatusPurpose) error {
	if !request.IsValid() {
		return util.LoggingNewError("store request request is not valid")
	}

	// transform the credential into its denormalized form for storage
	storedCredential, err := buildStoredCredential(request)
	if err != nil {
		return errors.Wrap(err, "building stored credential")
	}

	storedCredBytes, err := json.Marshal(storedCredential)
	if err != nil {
		return util.LoggingErrorMsgf(err, "could not store request: %s", storedCredential.CredentialID)
	}

	schemaID := ""
	if request.Credential.CredentialSchema != nil {
		schemaID = request.Credential.CredentialSchema.ID
	}

	statusListCredentialKey := createStatusListCredentialPrefixKey(storedCredential.CredentialID, request.Credential.Issuer.(string), request.Credential.CredentialSubject.GetID(), schemaID, string(statusPurpose))
	return tx.Write(ctx, statusListCredentialNamespace, statusListCredentialKey, storedCredBytes)
}

func (cs *Storage) GetStatusListCredential(ctx context.Context, id string) (*StoredCredential, error) {
	prefixValues, err := cs.db.ReadPrefix(ctx, statusListCredentialNamespace, id)
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not get credential from storage: %s", id)
	}
	if len(prefixValues) > 1 {
		return nil, util.LoggingNewErrorf("could not get credential from storage; multiple prefix values matched credential id: %s", id)
	}

	// since we know the map now only has a single value, we break after the first element
	var credBytes []byte
	for _, v := range prefixValues {
		credBytes = v
		break
	}
	if len(credBytes) == 0 {
		return nil, util.LoggingNewErrorf("could not get credential from storage %s with id: %s", credentialNotFoundErrMsg, id)
	}

	var stored StoredCredential
	if err = json.Unmarshal(credBytes, &stored); err != nil {
		return nil, util.LoggingErrorMsgf(err, "unmarshalling stored credential: %s", id)
	}
	return &stored, nil
}

func (cs *Storage) storeCredential(ctx context.Context, request StoreCredentialRequest, namespace string) error {

	wc, err := cs.getStoreCredentialWriteContext(request, namespace)
	if err != nil {
		return errors.Wrap(err, "building stored credential")

	}
	// TODO(gabe) conflict checking?
	return cs.db.Write(ctx, wc.namespace, wc.key, wc.value)
}

func (cs *Storage) GetStoreCredentialWriteContext(request StoreCredentialRequest) (*WriteContext, error) {
	return cs.getStoreCredentialWriteContext(request, credentialNamespace)
}

func (cs *Storage) getStoreCredentialWriteContext(request StoreCredentialRequest, namespace string) (*WriteContext, error) {
	if !request.IsValid() {
		return nil, util.LoggingNewError("store request request is not valid")
	}

	// transform the credential into its denormalized form for storage
	storedCredential, err := buildStoredCredential(request)
	if err != nil {
		return nil, errors.Wrap(err, "building stored credential")
	}

	storedCredBytes, err := json.Marshal(storedCredential)
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not store request: %s", storedCredential.CredentialID)
	}

	wc := WriteContext{
		namespace: namespace,
		key:       storedCredential.ID,
		value:     storedCredBytes,
	}

	return &wc, nil
}

// buildStoredCredential generically parses a store credential request and returns the object to be stored
func buildStoredCredential(request StoreCredentialRequest) (*StoredCredential, error) {
	// assume we have a Data Integrity credential
	cred := request.Credential
	if request.HasJWTCredential() {
		parsedCred, err := signing.ParseVerifiableCredentialFromJWT(request.CredentialJWT.String())
		if err != nil {
			return nil, errors.Wrap(err, "could not parse credential from jwt")
		}

		// if we have a JWT credential, update the reference
		cred = parsedCred
	}

	credID := cred.ID
	// Note: we assume the issuer is always a string for now
	issuer := cred.Issuer.(string)
	subject := cred.CredentialSubject.GetID()

	// schema is not a required field, so we must do this check
	schema := ""
	if cred.CredentialSchema != nil {
		schema = cred.CredentialSchema.ID
	}
	return &StoredCredential{
		ID:            createPrefixKey(credID, issuer, subject, schema),
		CredentialID:  credID,
		Credential:    cred,
		CredentialJWT: request.CredentialJWT,
		Issuer:        issuer,
		Subject:       subject,
		Schema:        schema,
		IssuanceDate:  cred.IssuanceDate,
		Revoked:       request.Revoked,
		Suspended:     request.Suspended,
	}, nil
}

func (cs *Storage) GetCredential(ctx context.Context, id string) (*StoredCredential, error) {
	log.Default().Println("Storage GetCredential Start Time ", time.Now().String())
	return cs.getCredential(ctx, id, credentialNamespace)
}

func (cs *Storage) getCredential(ctx context.Context, id string, namespace string) (*StoredCredential, error) {
	// log.Default().Println("cs.GetCredential Start Time ", time.Now().String())
	// prefixValues, err := cs.db.ReadPrefix(ctx, namespace, id)
	// if err != nil {
	// 	return nil, util.LoggingErrorMsgf(err, "could not get credential from storage: %s", id)
	// }
	// if len(prefixValues) > 1 {
	// 	return nil, util.LoggingNewErrorf("could not get credential from storage; multiple prefix values matched credential id: %s", id)
	// }

	// // since we know the map now only has a single value, we break after the first element
	// var credBytes []byte
	// for _, v := range prefixValues {
	// 	credBytes = v
	// 	break
	// }
	// if len(credBytes) == 0 {
	// 	return nil, util.LoggingNewErrorf("could not get credential from storage %s with id: %s", credentialNotFoundErrMsg, id)
	// }

	credbytes, err := cs.db.Read(ctx, "credential", "89370588-7e0c-4aa5-bf28-f6ee2e80b06e-is:did:key:z6Mkkhk1xKGZcgScbEg9qpFbV9ZEg13nTGh6soS7Z1R9V24v-su:did:key:z6MktqDyivH6vS7p6iRUZyT42ecEdWYso1eUw2UTn6KS8C7f-sc:ad143eea-ff01-4ee8-bd70-58fdcc77653")
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not get credential from storage: %s", id)
	}

	log.Default().Println("credentail value ", string(credbytes))

	var stored StoredCredential
	if err = json.Unmarshal(credbytes, &stored); err != nil {
		return nil, util.LoggingErrorMsgf(err, "unmarshalling stored credential: %s", id)
	}
	log.Default().Println("cs.GetCredential End Time ", time.Now().String())
	return &stored, nil
}

// Note: this is a lazy  implementation. Optimizations are to be had by adjusting prefix
// queries, and nested buckets. It is not intended that bolt is run in production, or at any scale,
// so this is not much of a concern.

// GetCredentialsByIssuer gets all credentials stored with a prefix key containing the issuer value
// The method is greedy, meaning if multiple values are found and some fail during processing, we will
// return only the successful values and log an error for the failures.
func (cs *Storage) GetCredentialsByIssuer(ctx context.Context, issuer string) ([]StoredCredential, error) {
	keys, err := cs.db.ReadAllKeys(ctx, credentialNamespace)
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not read credential storage while searching for creds for issuer: %s", issuer)
	}
	// see if the prefix keys contains the issuer value
	var issuerKeys []string
	for _, k := range keys {
		if strings.Contains(k, issuer) {
			issuerKeys = append(issuerKeys, k)
		}
	}
	if len(issuerKeys) == 0 {
		logrus.Warnf("no credentials found for issuer: %s", util.SanitizeLog(issuer))
		return nil, nil
	}

	// now get each credential by key
	var storedCreds []StoredCredential
	for _, key := range issuerKeys {
		credBytes, err := cs.db.Read(ctx, credentialNamespace, key)
		if err != nil {
			logrus.WithError(err).Errorf("could not read credential with key: %s", key)
		} else {
			var cred StoredCredential
			if err = json.Unmarshal(credBytes, &cred); err != nil {
				logrus.WithError(err).Errorf("unmarshalling credential with key: %s", key)
			}
			storedCreds = append(storedCreds, cred)
		}
	}

	if len(storedCreds) == 0 {
		logrus.Warnf("no credentials able to be retrieved for issuer: %s", issuerKeys)
	}

	return storedCreds, nil
}

// GetCredentialsBySubject gets all credentials stored with a prefix key containing the subject value
// The method is greedy, meaning if multiple values are found...and some fail during processing, we will
// return only the successful values and log an error for the failures.
func (cs *Storage) GetCredentialsBySubject(ctx context.Context, subject string) ([]StoredCredential, error) {
	keys, err := cs.db.ReadAllKeys(ctx, credentialNamespace)
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not read credential storage while searching for creds for subject: %s", subject)
	}

	// see if the prefix keys contains the subject value
	var subjectKeys []string
	for _, k := range keys {
		if strings.Contains(k, subject) {
			subjectKeys = append(subjectKeys, k)
		}
	}
	if len(subjectKeys) == 0 {
		logrus.Warnf("no credentials found for subject: %s", util.SanitizeLog(subject))
		return nil, nil
	}

	// now get each credential by key
	var storedCreds []StoredCredential
	for _, key := range subjectKeys {
		credBytes, err := cs.db.Read(ctx, credentialNamespace, key)
		if err != nil {
			logrus.WithError(err).Errorf("could not read credential with key: %s", key)
		} else {
			var cred StoredCredential
			if err := json.Unmarshal(credBytes, &cred); err != nil {
				logrus.WithError(err).Errorf("unmarshalling credential with key: %s", key)
			}
			storedCreds = append(storedCreds, cred)
		}
	}

	if len(storedCreds) == 0 {
		logrus.Warnf("no credentials able to be retrieved for subject: %s", subjectKeys)
	}

	return storedCreds, nil
}

// GetCredentialsBySchema gets all credentials stored with a prefix key containing the schema value
// The method is greedy, meaning if multiple values are found...and some fail during processing, we will
// return only the successful values and log an error for the failures.
func (cs *Storage) GetCredentialsBySchema(ctx context.Context, schema string) ([]StoredCredential, error) {
	keys, err := cs.db.ReadAllKeys(ctx, credentialNamespace)
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not read credential storage while searching for creds for schema: %s", schema)
	}

	// see if the prefix keys contains the schema value
	query := "sc:" + schema
	var schemaKeys []string
	for _, k := range keys {
		if strings.HasSuffix(k, query) {
			schemaKeys = append(schemaKeys, k)
		}
	}
	if len(schemaKeys) == 0 {
		logrus.Warnf("no credentials found for schema: %s", util.SanitizeLog(schema))
		return nil, nil
	}

	// now get each credential by key
	var storedCreds []StoredCredential
	for _, key := range schemaKeys {
		credBytes, err := cs.db.Read(ctx, credentialNamespace, key)
		if err != nil {
			logrus.WithError(err).Errorf("could not read credential with key: %s", key)
		} else {
			var cred StoredCredential
			if err := json.Unmarshal(credBytes, &cred); err != nil {
				logrus.WithError(err).Errorf("unmarshalling credential with key: %s", key)
			}
			storedCreds = append(storedCreds, cred)
		}
	}

	if len(storedCreds) == 0 {
		logrus.Warnf("no credentials able to be retrieved for schema: %s", schemaKeys)
	}

	return storedCreds, nil
}

// GetCredentialsByIssuerAndSchema gets all credentials stored with a prefix key containing the issuer value
// The method is greedy, meaning if multiple values are found...and some fail during processing, we will
// return only the successful values and log an error for the failures.
func (cs *Storage) GetCredentialsByIssuerAndSchema(ctx context.Context, issuer string, schema string) ([]StoredCredential, error) {
	return cs.getCredentialsByIssuerAndSchema(ctx, issuer, schema, credentialNamespace)
}

func (cs *Storage) GetStatusListCredentialsByIssuerAndSchema(ctx context.Context, statusPurpose statussdk.StatusPurpose, issuer string, schema string) ([]StoredCredential, error) {
	keys, err := cs.db.ReadAllKeys(ctx, statusListCredentialNamespace)
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not read credential storage while searching for creds for issuer: %s", issuer)
	}

	query := "sc:" + schema + "-sp:" + string(statusPurpose)
	var issuerSchemaKeys []string
	for _, k := range keys {
		if strings.Contains(k, issuer) && strings.HasSuffix(k, query) {
			issuerSchemaKeys = append(issuerSchemaKeys, k)
		}
	}

	if len(issuerSchemaKeys) == 0 {
		logrus.Warnf("no credentials found for issuer: %s and schema %s", util.SanitizeLog(issuer), util.SanitizeLog(schema))
		return nil, nil
	}

	// now get each credential by key
	var storedCreds []StoredCredential
	for _, key := range issuerSchemaKeys {
		credBytes, err := cs.db.Read(ctx, statusListCredentialNamespace, key)
		if err != nil {
			logrus.WithError(err).Errorf("could not read credential with key: %s", key)
		} else {
			var cred StoredCredential
			if err = json.Unmarshal(credBytes, &cred); err != nil {
				logrus.WithError(err).Errorf("unmarshalling credential with key: %s", key)
			}

			storedCreds = append(storedCreds, cred)
		}
	}

	if len(storedCreds) == 0 {
		logrus.Warnf("no credentials able to be retrieved for issuer: %s", issuerSchemaKeys)
	}

	return storedCreds, nil
}

func (cs *Storage) getCredentialsByIssuerAndSchema(ctx context.Context, issuer string, schema string, namespace string) ([]StoredCredential, error) {
	keys, err := cs.db.ReadAllKeys(ctx, namespace)
	if err != nil {
		return nil, util.LoggingErrorMsgf(err, "could not read credential storage while searching for creds for issuer: %s", issuer)
	}

	query := "sc:" + schema
	var issuerSchemaKeys []string
	for _, k := range keys {
		if strings.Contains(k, issuer) && strings.HasSuffix(k, query) {
			issuerSchemaKeys = append(issuerSchemaKeys, k)
		}
	}

	if len(issuerSchemaKeys) == 0 {
		logrus.Warnf("no credentials found for issuer: %s and schema %s", util.SanitizeLog(issuer), util.SanitizeLog(schema))
		return nil, nil
	}

	// now get each credential by key
	var storedCreds []StoredCredential
	for _, key := range issuerSchemaKeys {
		credBytes, err := cs.db.Read(ctx, namespace, key)
		if err != nil {
			logrus.WithError(err).Errorf("could not read credential with key: %s", key)
		} else {
			var cred StoredCredential
			if err = json.Unmarshal(credBytes, &cred); err != nil {
				logrus.WithError(err).Errorf("unmarshalling credential with key: %s", key)
			}
			storedCreds = append(storedCreds, cred)
		}
	}

	if len(storedCreds) == 0 {
		logrus.Warnf("no credentials able to be retrieved for issuer: %s", issuerSchemaKeys)
	}

	return storedCreds, nil
}

func (cs *Storage) DeleteCredential(ctx context.Context, id string) error {
	return cs.deleteCredential(ctx, id, credentialNamespace)
}

func (cs *Storage) DeleteStatusListCredential(ctx context.Context, id string) error {
	return cs.deleteCredential(ctx, id, statusListCredentialNamespace)
}

func (cs *Storage) deleteCredential(ctx context.Context, id string, namespace string) error {
	credDoesNotExistMsg := fmt.Sprintf("credential does not exist, cannot delete: %s", id)

	// first get the credential to regenerate the prefix key
	gotCred, err := cs.GetCredential(ctx, id)
	if err != nil {
		// no error on deletion for a non-existent credential
		if strings.Contains(err.Error(), credentialNotFoundErrMsg) {
			logrus.Warn(credDoesNotExistMsg)
			return nil
		}

		return util.LoggingErrorMsgf(err, "could not get credential<%s> before deletion", id)
	}

	// no error on deletion for a non-existent credential
	if gotCred == nil {
		logrus.Warn(credDoesNotExistMsg)
		return nil
	}

	// re-create the prefix key to delete
	prefix := createPrefixKey(id, gotCred.Issuer, gotCred.Subject, gotCred.Schema)
	if err = cs.db.Delete(ctx, namespace, prefix); err != nil {
		return util.LoggingErrorMsgf(err, "could not delete credential: %s", id)
	}
	return nil
}

func (cs *Storage) GetIncrementStatusListIndexWriteContextWatchKey() storage.WatchKey {
	return storage.WatchKey{Namespace: statusListIndexNamespace, Key: currentListIndexKey}
}

func (cs *Storage) GetStatusListIndexesWriteKey() storage.WatchKey {
	return storage.WatchKey{Namespace: statusListIndexNamespace, Key: statusListIndexesKey}
}

// unique key for a credential
func createPrefixKey(id, issuer, subject, schema string) string {
	return strings.Join([]string{id, "is:" + issuer, "su:" + subject, "sc:" + schema}, "-")
}

func createStatusListCredentialPrefixKey(id, issuer, subject, schema, statusPurpose string) string {
	return strings.Join([]string{id, "is:" + issuer, "su:" + subject, "sc:" + schema, "sp:" + statusPurpose}, "-")
}

func randomUniqueNum(count int) []int {
	randomNumbers := make([]int, 0, count)

	for i := 1; i <= count; i++ {
		randomNumbers = append(randomNumbers, i)
	}

	rand.Shuffle(len(randomNumbers), func(i, j int) {
		randomNumbers[i], randomNumbers[j] = randomNumbers[j], randomNumbers[i]
	})

	return randomNumbers
}
