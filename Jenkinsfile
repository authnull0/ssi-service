pipeline {
    agent any
    triggers {
        githubPush()
    }
    options {
        buildDiscarder logRotator(artifactDaysToKeepStr: '', artifactNumToKeepStr: '10', daysToKeepStr: '', numToKeepStr: '10')
    }
    stages {
        stage('Checkout') {
            steps {
            git credentialsId: 'amanpd-github-credentials', url: 'https://github.com/authnull0/ssi-service.git', branch: 'persistance-fix'
            }
        }
        stage('Build Docker Image') {
            steps {
                script {
                    def dockerImage = "ssi-service:${env.BUILD_ID}"
                    env.dockerImage = dockerImage
                    sh "docker build . --tag ${dockerImage}"
                    echo "Docker Image Name: ${dockerImage}"
                    echo "${env.BUILD_ID}"
                }
            }
        }
        stage('Stop & Remove older image') {
            steps {
                script {
                    def previousBuildNumber = env.BUILD_ID.toInteger() - 1
                    def dockerCommand = "ssi-service-${previousBuildNumber}"
                    if (sh(script: "docker ps -a | grep ${dockerCommand}", returnStatus: true) == 0) {
                        sh "docker stop ${dockerCommand}"
                        sh "docker rm ${dockerCommand}"
                    }
                }
            }
        }
        stage('Run Docker Container') {
            steps {
                script {
                    def dockerCommand = "-d -p 5000:5000 --name=ssi-service-${env.BUILD_ID} ${env.dockerImage}"
                    sh "docker run ${dockerCommand}"
                }
            }
        }
    }
}
