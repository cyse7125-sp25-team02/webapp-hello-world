pipeline {
    agent any
    
    environment {
        DOCKERHUB_CREDENTIALS = credentials('docker-hub-credentials')
        DOCKER_IMAGE = "karanthakkar09/webapp-hello-world"
    }
    
    stages {
        stage('Prepare Workspace and Determine Next Version') {
            steps {
                script {
                    cleanWs()
                    sh 'git config --global user.email "jenkins@jkops.com"'
                    sh 'git config --global user.name "Jenkins"'
                    
                    git branch: 'master', url: 'https://github.com/cyse7125-sp25-team02/webapp-hello-world', credentialsId: 'github-credentials'
                    env.NEXT_VERSION = nextVersion()
                }
            }
        }

        stage('Push New Tag Version') {
            steps {
                script {
                    sh "git tag -a ${env.NEXT_VERSION} -m 'Release version ${env.NEXT_VERSION}'"
                    withCredentials([gitUsernamePassword(credentialsId: 'github-credentials', gitToolName: 'Default')]) {
                        sh "git push origin ${env.NEXT_VERSION}"
                    }
                }
            }
        }

        stage('Setup BuildX') {
            steps {
                sh '''
                    docker buildx create --use
                    docker buildx inspect --bootstrap
                '''
            }
        }
        
        stage('Login and Build') {
            steps {
                sh 'echo $DOCKERHUB_CREDENTIALS_PSW | docker login -u $DOCKERHUB_CREDENTIALS_USR --password-stdin'

                script {
                    sh """
                        docker buildx build --platform linux/amd64,linux/arm64 \
                        -t ${DOCKER_IMAGE}:latest \
                        -t ${DOCKER_IMAGE}:${env.NEXT_VERSION} \
                        --push .
                    """
                }
            }
        }
    }
    
    post {
        always {
            sh 'docker logout'
            sh 'docker builder prune -f'
            cleanWs()
        }
        success {
            echo "Successfully built and published Docker image ${DOCKER_IMAGE}:${env.NEXT_VERSION}"
        }
    }
}
