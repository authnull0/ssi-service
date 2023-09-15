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
            git credentialsId: 'amanpd-github-credentials', url: 'https://github.com/authnull0/ssi-service.git', branch: 'development'
            }
        }
        stage('Sonarqube Scanning') {
            environment {
                scannerHome = tool 'SonarQubeScanner'
                scannerCmd = "${scannerHome}/bin/sonar-scanner"
                scannerCmdOptions = "-Dsonar.projectKey=ssi-service -Dsonar.sources=build,cmd,config,doc,gui,integration,internal,pkg,sip -Dsonar.host.url=http://195.201.165.12:9000"
                }
            steps {
                withSonarQubeEnv(installationName: 'sonarqube-server') {
                sh "${scannerCmd} ${scannerCmdOptions}"
                }
                timeout(time: 10, unit: 'MINUTES') {
                waitForQualityGate abortPipeline: true
                }
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
        stage('Trivy Scan Docker Image') {
            steps {
                script {
                    def formatOption = "--format template --template \"@/usr/local/share/trivy/templates/html.tpl\""
                    sh """
                    trivy image ${env.dockerImage} $formatOption --timeout 10m --output report.html || true
                    """
            }
        publishHTML(target: [
          allowMissing: true,
          alwaysLinkToLastBuild: false,
          keepAll: true,
          reportDir: ".",
          reportFiles: "report.html",
          reportName: "Trivy Report",
        ])
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
                    def dockerCommand = "-d --network host --name=ssi-service-${env.BUILD_ID} ${env.dockerImage}"
                    sh "docker run ${dockerCommand}"
                }
            }
        }
    }
}
