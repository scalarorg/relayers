pipeline {
    agent any
    options {
        skipStagesAfterUnstable()
    }
    stages {
        stage('Preparation') {
            steps {
                echo "env:  ${env.getEnvironment()}"
            }
        }
        stage('Clone') {
            steps {
                echo 'Cloning relayers'
                sh 'git clone https://github.com/scalarorg/relayers.git'
            }
        }
        stage('Build') {
            steps {
                sh 'make docker-image'
            }
        }
        stage('Start'){
            steps {
                sh 'task -t ~/tasks/e2e.yml relayer:up'
            }
        }
        stage('Bridging') {
            steps {
                sh 'task -t ~/tasks/e2e.yml bridge:pooling'
                sh 'task -t ~/tasks/e2e.yml bridge:upc'
            }
        }
        stage('Bridging verification') {
            steps {
                echo 'Bridging verification'
            }
        }
    }
}