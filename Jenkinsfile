pipeline {
    agent any
    options {
        skipStagesAfterUnstable()
    }
    stages {
        stage('Build') {
            steps {
                sh 'make docker-image-test'
            }
        }
        stage('Start scalar network'){
            steps {
                sh 'task -t ~/tasks/e2e.yml scalar:up'
                sh 'task -t ~/tasks/e2e.yml scalar:multisig'
                sh 'task -t ~/tasks/e2e.yml scalar:token-deploy'
            }
        }
         stage('Start relayer'){
            steps {
                // sh 'docker ps -a --format "{{.Names}}" | grep -w "scalar-relayer" && docker rm -f scalar-relayer'
                sh 'export IMAGE_TAG_RELAYER=$(git log -1 --format="%H") && task -t ~/tasks/e2e.yml relayer:up'
            }
        }
        stage('Bridging') {
            steps {
                echo 'Bridging task'
                // sh 'task -t ~/tasks/e2e.yml bridge:pooling'
                // sh 'task -t ~/tasks/e2e.yml bridge:upc'
            }
        }
        stage('Bridging verification') {
            steps {
                echo 'Bridging verification'
            }
        }
    }
}