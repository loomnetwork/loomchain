// Inspired from https://jenkins.io/doc/pipeline/examples/

def labels = ['linux', 'windows', 'osx'] // labels for Jenkins node types we will build on

stages {
  stage ('Checkout') {
    def checkouts = [:]
    for (x in labels) {
      def label = x
      
      checkouts[label] = {
        node[label] {
            def scmVars = checkout scm
            def commitHash = checkout(scm).GIT_COMMIT
        }
      }
    }
    
    parallel checkouts
  }
  
  stage ('Build') {
    def builders = [:]
    for (x in labels) {
      def label = x // Need to bind the label variable before the closure - can't do 'for (label in labels)'

      // Create a map to pass in to the 'parallel' step so we can fire all the builds at once
      builders[label] = {
        node(label) {
          if (label!='windows') {
            sh '''
                ls -l
                pwd
                ./jenkins.sh
            '''
          } else {
            sh '''
                ./jenkins.bat
            '''
          }
        }
      }
    }

    parallel builders
  }
}
