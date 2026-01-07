pipeline {
    agent any

    environment {
        GITHUB_REPO = 'https://github.com/authnull0/ssi-service.git'
        GITHUB_BRANCH = 'production-az'
        PRIVATE_TAG = 'production'
        PUBLIC_TAG  = '1.0.0'
        DOCKER_REGISTRY_PRIVATE = 'docker-repo.authnull.com'
        DOCKER_REGISTRY_PUBLIC= 'docker-repo-public.authnull.com'
        DOCKER_PRIVATE_CREDENTIALS = credentials('authnull-repo')
        DOCKER_PUBLIC_CREDENTIALS = credentials('docker-repo-public')
        DOCKER_IMAGE_PRIVATE = "docker-repo.authnull.com/ssi-service:${PRIVATE_TAG}"
        DOCKER_IMAGE_PUBLIC = "docker-repo-public.authnull.com/ssi-service:${PUBLIC_TAG}"
//        SONARQUBE_SERVER = 'Sonar-Qube-servers'  
//        SONARQUBE_PROJECT_KEY = 'Authnullproject'  
//        SONAR_HOST_URL = 'https://scan.authnull.com/' 
//        SONAR_AUTH_TOKEN = credentials('sonar-auth-token')
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
//        stage('SonarQube Analysis') {
//            steps {
//                script {
//                    // Reference the SonarQube scanner tool installed on Jenkins
//                    def scannerHome = tool name: 'SonarQube Scanner 4.7'
//                    withSonarQubeEnv("${SONARQUBE_SERVER}") {
//                        sh """
//                            ${scannerHome}/sonar-scanner \
//                                -Dsonar.projectKey=${SONARQUBE_PROJECT_KEY} \
//                                -Dsonar.sources=. \
//                                -Dsonar.host.url=${SONAR_HOST_URL} \
//                                -Dsonar.login=${SONAR_AUTH_TOKEN}
//                        """
//                    }
//                }
//            }
//        }
                // Build Stage

        stage('Build Private Image') {
            steps {
                sh """
                    echo "Building PRIVATE image: ${DOCKER_IMAGE_PRIVATE}"
                    docker build -t ${DOCKER_IMAGE_PRIVATE} .
                """
            }
        }

        stage('Build Public Image') {
            steps {
                sh """
                    echo "Building PUBLIC image: ${DOCKER_IMAGE_PUBLIC}"
                    docker build -t ${DOCKER_IMAGE_PUBLIC} .
                """
            }
        }

        // Push Stage

        stage('Push Private Image') {
            steps {
                withCredentials([usernamePassword(credentialsId: 'authnull-repo', usernameVariable: 'USR', passwordVariable: 'PASS')]) {
                    sh """
                        echo "Logging in to PRIVATE registry"
                        echo "\$PASS" | docker login ${DOCKER_REGISTRY_PRIVATE} -u "\$USR" --password-stdin

                        docker push ${DOCKER_IMAGE_PRIVATE}
                        docker logout ${DOCKER_REGISTRY_PRIVATE}
                    """
                }
            }
        }

        stage('Push Public Image') {
            steps {
                withCredentials([usernamePassword(credentialsId: 'docker-repo-public', usernameVariable: 'USR', passwordVariable: 'PASS')]) {
                    sh """
                        echo "Logging in to PUBLIC registry"
                        echo "\$PASS" | docker login ${DOCKER_REGISTRY_PUBLIC} -u "\$USR" --password-stdin

                        docker push ${DOCKER_IMAGE_PUBLIC}
                        docker logout ${DOCKER_REGISTRY_PUBLIC}
                    """
                }
            }
        }

        // Cleanup Stage

        stage('Cleanup Images') {
            steps {
                sh """
                    docker rmi ${DOCKER_IMAGE_PRIVATE} || true
                    docker rmi ${DOCKER_IMAGE_PUBLIC} || true
                """
            }
        }
    }
}