pipeline {
    agent any

    environment {
        GITHUB_REPO = 'https://github.com/authnull0/ssi-service.git'
        GITHUB_BRANCH = 'onprem'
        PUBLIC_TAG  = '1.0.0-onprem'
        DOCKER_REGISTRY_PUBLIC= 'docker-repo-public.authnull.com'
        DOCKER_PUBLIC_CREDENTIALS = credentials('docker-repo-public')
        DOCKER_IMAGE_PUBLIC = "docker-repo-public.authnull.com/ssi-service:${PUBLIC_TAG}"
        GITHUB_TOKEN = credentials('ram-Github-credentials')
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

        stage('Build Public Image') {
            steps {
                sh """
                    echo "Building PUBLIC image: ${DOCKER_IMAGE_PUBLIC}"
                    docker build -t ${DOCKER_IMAGE_PUBLIC} .
                """
            }
        }

        // Push Stage

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
                    docker rmi ${DOCKER_IMAGE_PUBLIC} || true
                """
            }
        }
    }
}
