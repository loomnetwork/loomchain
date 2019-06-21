void setBuildStatus(String message, String state, String context) {
  step([
      $class: "GitHubCommitStatusSetter",
      reposSource: [$class: "ManuallyEnteredRepositorySource", url: "git@github.com:loomnetwork/loomchain.git"],
      contextSource: [$class: "ManuallyEnteredCommitContextSource", context: context],
      errorHandlers: [[$class: "ChangingBuildStatusErrorHandler", result: "UNSTABLE"]],
      statusResultSource: [ $class: "ConditionalStatusResultSource", results: [[$class: "AnyBuildResult", message: message, state: state]] ]
  ]);
}

def builders = [:]
def disabled = [:]

builders['linux'] = {
  node('linux-any') {
    timestamps {
      def thisBuild = null

      try {
        stage ('Checkout - Linux') {
          checkout changelog: true, poll: true, scm:
          [
            $class: 'GitSCM',
            branches: [[name: 'refs/heads/master']],
            doGenerateSubmoduleConfigurations: false,
            extensions: [[$class: 'CleanBeforeCheckout'], [$class: 'PruneStaleBranch']],
            submoduleCfg: [],
            userRemoteConfigs:
            [[
              credentialsId: 'loom-sdk',
              url: 'git@github.com:loomnetwork/loomchain.git'
            ]]
          ]
        }

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Linux");

        stage ('Build - Linux') {
          sh '''
            ./jenkins.sh
            cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
            gsutil cp loom-generic gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loom
            gsutil cp loom-gateway gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loom-gateway
            gsutil cp loom-cleveldb gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loom-cleveldb
            gsutil cp plasmachain gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/plasmachain
            gsutil cp plasmachain-cleveldb gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/plasmachain-cleveldb
            gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/validators-tool
            gsutil cp tgoracle gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/tgoracle
            gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loomcoin_tgoracle
            gsutil cp tron_tgoracle gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/tron_tgoracle
            gsutil cp dposv2_oracle gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/dposv2_oracle
            gsutil cp loom-generic gs://private.delegatecall.com/loom/linux/latest/loom
            gsutil cp loom-gateway gs://private.delegatecall.com/loom/linux/latest/loom-gateway
            gsutil cp loom-cleveldb gs://private.delegatecall.com/loom/linux/latest/loom-cleveldb
            gsutil cp plasmachain gs://private.delegatecall.com/loom/linux/latest/plasmachain
            gsutil cp plasmachain-cleveldb gs://private.delegatecall.com/loom/linux/latest/plasmachain-cleveldb
            gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/linux/latest/validators-tool
            gsutil cp tgoracle gs://private.delegatecall.com/loom/linux/latest/tgoracle
            gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/linux/latest/loomcoin_tgoracle
            gsutil cp tron_tgoracle gs://private.delegatecall.com/loom/linux/latest/tron_tgoracle
            gsutil cp dposv2_oracle gs://private.delegatecall.com/loom/linux/latest/dposv2_oracle
            gsutil cp install.sh gs://private.delegatecall.com/install.sh
            docker build --build-arg BUILD_NUMBER=${BUILD_NUMBER} -t loomnetwork/loom:latest .
            docker tag loomnetwork/loom:latest loomnetwork/loom:${BUILD_NUMBER}
            docker push loomnetwork/loom:latest
            docker push loomnetwork/loom:${BUILD_NUMBER}
          '''
        }
      } catch (e) {
        thisBuild = 'FAILURE'
        throw e
      } finally {
        if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Linux");
          slackSend channel: '#blockchain-engineers', color: '#FF0000', message: "${env.JOB_NAME} (LINUX) - #${env.BUILD_NUMBER} Failure after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
          sh '''
            cd /tmp/gopath-jenkins-${JOB_BASE_NAME}-${BUILD_NUMBER}/src/github.com/loomnetwork/loomchain/e2e
            find test-data -name "*.log" | tar -czf ${JOB_BASE_NAME}-${BUILD_NUMBER}-linux-test-data.tar.gz -T -
            mkdir -p /tmp/test-data
            mv ${JOB_BASE_NAME}-${BUILD_NUMBER}-linux-test-data.tar.gz /tmp/test-data
          '''
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Linux");
          slackSend channel: '#blockchain-engineers', color: '#006400', message: "${env.JOB_NAME} (LINUX) - #${env.BUILD_NUMBER} Success after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
        }
      }
    }
  }
}

builders['osx'] = {
  node('osx-any') {
    timestamps {
      def thisBuild = null

      try {
        stage ('Checkout - OSX') {
          checkout changelog: true, poll: true, scm:
          [
            $class: 'GitSCM',
            branches: [[name: 'refs/heads/master']],
            doGenerateSubmoduleConfigurations: false,
            extensions: [[$class: 'CleanBeforeCheckout'], [$class: 'PruneStaleBranch']],
            submoduleCfg: [],
            userRemoteConfigs:
            [[
              credentialsId: 'loom-sdk',
              url: 'git@github.com:loomnetwork/loomchain.git'
            ]]
          ]
        }

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "OSX");

        stage ('Build - OSX') {
          sh '''
            ./jenkins.sh
            cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
            gsutil cp loom-generic gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/loom
            gsutil cp loom-gateway gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/loom-gateway
            gsutil cp loom-cleveldb gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/loom-cleveldb
            gsutil cp plasmachain gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/plasmachain
            gsutil cp plasmachain-cleveldb gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/plasmachain-cleveldb
            gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/validators-tool
            gsutil cp tgoracle gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/tgoracle
            gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/loomcoin_tgoracle
            gsutil cp tron_tgoracle gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/tron_tgoracle
            gsutil cp dposv2_oracle gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/dposv2_oracle
            gsutil cp loom-generic gs://private.delegatecall.com/loom/osx/latest/loom
            gsutil cp loom-gateway gs://private.delegatecall.com/loom/osx/latest/loom-gateway
            gsutil cp loom-cleveldb gs://private.delegatecall.com/loom/osx/latest/loom-cleveldb
            gsutil cp plasmachain gs://private.delegatecall.com/loom/osx/latest/plasmachain
            gsutil cp plasmachain-cleveldb gs://private.delegatecall.com/loom/osx/latest/plasmachain-cleveldb
            gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/osx/latest/validators-tool
            gsutil cp tgoracle gs://private.delegatecall.com/loom/osx/latest/tgoracle
            gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/osx/latest/loomcoin_tgoracle
            gsutil cp tron_tgoracle gs://private.delegatecall.com/loom/osx/latest/tron_tgoracle
            gsutil cp dposv2_oracle gs://private.delegatecall.com/loom/osx/latest/dposv2_oracle
          '''
        }
      } catch (e) {
        thisBuild = 'FAILURE'
        throw e
      } finally {
        if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "OSX");
          slackSend channel: '#blockchain-engineers', color: '#FF0000', message: "${env.JOB_NAME} (OSX) - #${env.BUILD_NUMBER} Failure after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
          sh '''
            cd /tmp/gopath-jenkins-${JOB_BASE_NAME}-${BUILD_NUMBER}/src/github.com/loomnetwork/loomchain/e2e
            find test-data -name "*.log" | tar -czf ${JOB_BASE_NAME}-${BUILD_NUMBER}-osx-test-data.tar.gz -T -
            mkdir -p /tmp/test-data
            mv ${JOB_BASE_NAME}-${BUILD_NUMBER}-osx-test-data.tar.gz /tmp/test-data
          '''
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "OSX");
          slackSend channel: '#blockchain-engineers', color: '#006400', message: "${env.JOB_NAME} (OSX) - #${env.BUILD_NUMBER} Success after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
        }
      }
      build job: 'homebrew-client', parameters: [[$class: 'StringParameterValue', name: 'LOOM_BUILD', value: "$BUILD_NUMBER"]]
    }
  }
}

throttle(['loom-sdk']) {
  timeout(time: 60, unit: 'MINUTES'){
    parallel builders
  }
}
