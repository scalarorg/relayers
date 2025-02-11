pipeline {
    agent any
    options {
        skipStagesAfterUnstable()
    }
    stages {
        stage('Build') {
            steps {
                sh 'make docker-image'
            }
        }
        stage('Start'){
            steps {
                task -t ~/tasks/e2e.yml relayer:up
            }
        }
        stage('Bridging') {
            steps {
                task -t ~/tasks/e2e.yml bridge:pooling
                task -t ~/tasks/e2e.yml bridge:upc
            }
        }
        stage('Bridging verification') {
            steps {
                echo 'Bridging verification'
            }
        }
    }
}