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
        stage('Start'){
            steps {
                sh 'task -t ~/tasks/e2e.yml scalar:up'
                sh 'task -t ~/tasks/e2e.yml relayer:up'
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