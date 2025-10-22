pipeline {
    agent any

    environment {
        GITHUB_REPO = 'https://github.com/authnull0/ssi-service.git'
        GITHUB_BRANCH = 'development-test'
        DOCKER_REGISTRY = 'docker-repo.authnull.com'
        DOCKER_REGISTRY_CREDENTIALS = credentials('authnull-repo')
        DOCKER_IMAGE = 'docker-repo.authnull.com/ssi-service:latest'
        SONARQUBE_SERVER = 'Sonar-Qube-servers'  
        SONARQUBE_PROJECT_KEY = 'ssi-service'  
        SONAR_HOST_URL = 'https://scan.authnull.com/' 
        SONAR_AUTH_TOKEN = credentials('sonar-auth-token')
        SERVICE_NAME = 'ssi-service'
        TEAMS_WEBHOOK_URL = credentials('teams-webhook-aipolicy')
        GITHUB_TOKEN = credentials('my-test-token')
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
                 checkout scm
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
                                -Dsonar.projectName=${SERVICE_NAME} \
                                -Dsonar.sources=. \
                                -Dsonar.host.url=${SONAR_HOST_URL} \
                                -Dsonar.login=${SONAR_AUTH_TOKEN} \
                                -Dsonar.sourceEncoding=UTF-8 \
                                -Dsonar.go.coverage.reportPaths=coverage.out \
                                -Dsonar.go.tests.reportPaths=test-report.xml \
                                -Dsonar.inclusions="**/*" \
                                -Dsonar.exclusions="**/*.md,.git/**" \
                                -Dsonar.scm.forceReloadAll=true \
                                -Dsonar.verbose=true \
                                -Dsonar.scm.disabled=false \
                                -Dsonar.scm.provider=git
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
    }
    post {
        always {
            script {
                def qualityGate = waitForQualityGate()
                def reportUrl = "${SONAR_HOST_URL}/dashboard?id=${SONARQUBE_PROJECT_KEY}"
            
                // Step 2: Create Teams MessageCard payload
                def payload = """
                {
                    "@type": "MessageCard",
                    "@context": "http://schema.org/extensions",
                    "themeColor": "${qualityGate.status == 'OK' ? '00FF00' : 'FF0000'}",
                    "summary": "SonarQube Report - ${SERVICE_NAME}",
                    "title": "${qualityGate.status == 'OK' ? '✅ PASSED' : '❌ FAILED'} - ${SERVICE_NAME}",
                    "text": "**Project:** ${SERVICE_NAME}\\n\\n**Quality Gate:** ${qualityGate.status}\\n\\n[View Full Report](${reportUrl})",
                    "potentialAction": [{
                        "@type": "OpenUri",
                        "name": "Open in SonarQube",
                        "targets": [{
                            "os": "default",
                            "uri": "${reportUrl}"
                        }]
                    }]
                }
                """

                // Debugging (optional)
                writeFile file: 'teams_payload.json', text: payload
                
                // Use withCredentials to safely handle the secret / send to teams
                withCredentials([string(credentialsId: 'teams-webhook-aipolicy', variable: 'TEAMS_WEBHOOK_URL')]) {
                    sh """
                        curl -X POST -H "Content-Type: application/json" \
                        -d @teams_payload.json \
                        ${TEAMS_WEBHOOK_URL}
                    """
                }
            }
        }
    }
}
