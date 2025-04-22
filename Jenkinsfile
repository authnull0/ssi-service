pipeline {
    agent any

    environment {
        GITHUB_REPO = 'https://github.com/authnull0/ssi-service.git'
        GITHUB_BRANCH = 'development-test'
        DOCKER_REGISTRY = 'docker-repo.authnull.com'
        DOCKER_REGISTRY_CREDENTIALS = credentials('authnull-repo')
        DOCKER_IMAGE = 'docker-repo.authnull.com/ssi-service:latest'
        SONARQUBE_SERVER = 'Sonar-Qube-servers'  
        SONARQUBE_PROJECT_KEY = 'Authnullproject'  
        SONAR_HOST_URL = 'https://scan.authnull.com/' 
        SONAR_AUTH_TOKEN = credentials('sonar-auth-token')
        AZURE_SUBSCRIPTION_ID = credentials('subscriptionId')
        AKS_CLUSTER = 'authnull-v2'
        RESOURCE_GROUP = 'azure-k8s'
        K8S_NAMESPACE = 'authnull-dev'
        AZURE_CLIENT_ID = credentials('clientid')
        AZURE_CLIENT_SECRET = credentials('secretid')
        AZURE_TENANT_ID = credentials('tenantid')
    }

    triggers {
        githubPush()
    }
    options {
        buildDiscarder logRotator(artifactDaysToKeepStr: '', artifactNumToKeepStr: '10', daysToKeepStr: '', numToKeepStr: '10')
    }
        
    stages {
        stage('Checkout') {
            steps {
            git credentialsId: 'pshussain-github', url: "${GITHUB_REPO}", branch: "${GITHUB_BRANCH}"
            }
        }
        stage('SonarQube Analysis') {
            steps {
                script {
                    // Reference the SonarQube scanner tool installed on Jenkins
                    def scannerHome = tool name: 'SonarQube Scanner 4.7'
                    withSonarQubeEnv("${SONARQUBE_SERVER}") {
                        sh """
                            ${scannerHome}/sonar-scanner \
                                -Dsonar.projectKey=${SONARQUBE_PROJECT_KEY} \
                                -Dsonar.sources=. \
                                -Dsonar.host.url=${SONAR_HOST_URL} \
                                -Dsonar.login=${SONAR_AUTH_TOKEN}
                        """
                    }
                }
            }
        }        
        stage('Build Docker Image') {
            steps {
                sh 'docker build -t ${DOCKER_IMAGE} .'
            }
        }
        stage('Login to Docker Artifactory') {
            steps {
                sh 'echo ${DOCKER_REGISTRY_CREDENTIALS_PSW} | docker login ${DOCKER_REGISTRY} -u ${DOCKER_REGISTRY_CREDENTIALS_USR} --password-stdin'
            }
        }
        stage('Push Docker Image') {
            steps {
                sh 'docker push ${DOCKER_IMAGE}'
            }
        }
        stage('Logout from Docker Artifactory') {
            steps {
                sh 'docker logout ${DOCKER_REGISTRY}'
            }
        }
        stage('Remove Docker Image') {
            steps {
                sh 'docker rmi ${DOCKER_IMAGE}'
            }
        }
        stage('Authenticate to AKS') {
             steps {
                sh '''
                    az login --service-principal -u "$AZURE_CLIENT_ID" -p "$AZURE_CLIENT_SECRET" --tenant "$AZURE_TENANT_ID"
                    az account set --subscription "$AZURE_SUBSCRIPTION_ID"
                    az aks get-credentials --resource-group "$RESOURCE_GROUP" --name "$AKS_CLUSTER"
                '''
            }
        }
        stage('Delete Existing Pods') {
            steps {
                sh 'kubectl delete pods -l app=dev-ssi-service-app -n ${K8S_NAMESPACE} --ignore-not-found'
            }
        }
    }
}
