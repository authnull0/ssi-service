pipeline {
    agent any

    environment {
        GITHUB_REPO = 'https://github.com/authnull0/ssi-service.git'
        GITHUB_BRANCH = 'production-az'
        DOCKER_REGISTRY = 'docker-repo.authnull.com'
        DOCKER_REGISTRY_CREDENTIALS = credentials('authnull-repo')
        DOCKER_IMAGE = 'docker-repo.authnull.com/ssi-service:production'
        SONARQUBE_SERVER = 'Sonar-Qube-servers'  
        SONARQUBE_PROJECT_KEY = 'Authnullproject'  
        SONAR_HOST_URL = 'https://scan.authnull.com/' 
        SONAR_AUTH_TOKEN = credentials('sonar-auth-token')
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
                 // ✅ Use Jenkins' built-in checkout which already uses the credentials from job config
                 checkout scm

                 // ✅ Force authenticated remote URL to fix anonymous fetch issue
                //sh """
                  //  git remote set-url origin https://my-test-token:${GITHUB_TOKEN}@github.com/authnull0/dashboard-service.git 
                    //git fetch origin ${GITHUB_BRANCH}
               // """
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
    }
}
