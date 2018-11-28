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
  node('linux') {
    def thisBuild = null

    try {
      stage ('Checkout - Linux') {
        checkout changelog: true, poll: true, scm:
        [
          $class: 'GitSCM',
          branches: [[name: 'refs/heads/v2']],
          doGenerateSubmoduleConfigurations: false,
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
          gsutil cp loom gs://private.delegatecall.com/loom/linux/v2-$BUILD_NUMBER/loom
          gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/linux/v2-$BUILD_NUMBER/validators-tool
          gsutil cp tgoracle gs://private.delegatecall.com/loom/linux/v2-$BUILD_NUMBER/tgoracle
          gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/linux/v2-$BUILD_NUMBER/loomcoin_tgoracle
          gsutil cp loom gs://private.delegatecall.com/loom/linux/v2-latest/loom
          gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/linux/v2-latest/validators-tool
          gsutil cp tgoracle gs://private.delegatecall.com/loom/linux/v2-latest/tgoracle
          gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/linux/v2-latest/loomcoin_tgoracle
          docker build --build-arg BUILD_NUMBER=${BUILD_NUMBER} -t loomnetwork/loom:v2-latest .
          docker tag loomnetwork/loom:v2-latest loomnetwork/loom:v2-${BUILD_NUMBER}
          docker push loomnetwork/loom:v2-latest
          docker push loomnetwork/loom:v2-${BUILD_NUMBER}
        '''
      }
    } catch (e) {
      thisBuild = 'FAILURE'
      throw e
    } finally {
      if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Linux");
        slackSend channel: '#blockchain-engineers', color: '#FF0000', message: "${env.JOB_NAME} (LINUX) - #${env.BUILD_NUMBER} Failure after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Linux");
        slackSend channel: '#blockchain-engineers', color: '#006400', message: "${env.JOB_NAME} (LINUX) - #${env.BUILD_NUMBER} Success after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
      }
    }
  }
}

disabled['windows'] = {
  node('windows') {
    def thisBuild = null

    try {
      stage ('Checkout - Windows') {
        checkout changelog: true, poll: true, scm:
        [
          $class: 'GitSCM',
          branches: [[name: 'refs/heads/v2']],
          doGenerateSubmoduleConfigurations: false,
          submoduleCfg: [],
          userRemoteConfigs:
          [[
            credentialsId: 'loom-sdk',
            url: 'git@github.com:loomnetwork/loomchain.git'
          ]]
        ]
      }

      setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Windows");

      stage ('Build - Windows') {
        bat '''
          jenkins.cmd
          SET PATH=C:\\Program Files (x86)\\Google\\Cloud SDK\\google-cloud-sdk\\bin;%PATH%;
          cd \\msys64\\tmp\\gopath-${BUILD_TAG}\\src\\github.com\\loomnetwork\\loomchain
          gsutil cp loom.exe gs://private.delegatecall.com/loom/windows/v2-$BUILD_NUMBER/loom.exe
          gsutil cp loom.exe gs://private.delegatecall.com/loom/windows/v2-latest/loom.exe
        '''
      }
    } catch (e) {
      thisBuild = 'FAILURE'
      throw e
    } finally {
      if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Windows");
        slackSend channel: '#blockchain-engineers', color: '#FF0000', message: "${env.JOB_NAME} (WINDOWS) - #${env.BUILD_NUMBER} Failure after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Windows");
        slackSend channel: '#blockchain-engineers', color: '#006400', message: "${env.JOB_NAME} (WINDOWS) - #${env.BUILD_NUMBER} Success after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
      }
    }
  }
}

builders['osx'] = {
  node('osx') {
    def thisBuild = null

    try {
      stage ('Checkout - OSX') {
        checkout changelog: true, poll: true, scm:
        [
          $class: 'GitSCM',
          branches: [[name: 'refs/heads/v2']],
          doGenerateSubmoduleConfigurations: false,
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
          gsutil cp loom gs://private.delegatecall.com/loom/osx/v2-$BUILD_NUMBER/loom
          gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/osx/v2-$BUILD_NUMBER/validators-tool
          gsutil cp tgoracle gs://private.delegatecall.com/loom/osx/v2-$BUILD_NUMBER/tgoracle
          gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/osx/v2-$BUILD_NUMBER/loomcoin_tgoracle
          gsutil cp loom gs://private.delegatecall.com/loom/osx/v2-latest/loom
          gsutil cp e2e/validators-tool gs://private.delegatecall.com/loom/osx/v2-latest/validators-tool
          gsutil cp tgoracle gs://private.delegatecall.com/loom/osx/v2-latest/tgoracle
          gsutil cp loomcoin_tgoracle gs://private.delegatecall.com/loom/osx/v2-latest/loomcoin_tgoracle
        '''
      }
    } catch (e) {
      thisBuild = 'FAILURE'
      throw e
    } finally {
      if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "OSX");
        slackSend channel: '#blockchain-engineers', color: '#FF0000', message: "${env.JOB_NAME} (OSX) - #${env.BUILD_NUMBER} Failure after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "OSX");
        slackSend channel: '#blockchain-engineers', color: '#006400', message: "${env.JOB_NAME} (OSX) - #${env.BUILD_NUMBER} Success after ${currentBuild.durationString.replace(' and counting', '')} (<${env.BUILD_URL}|Open>)"
      }
    }
    build job: 'homebrew-client', parameters: [[$class: 'StringParameterValue', name: 'LOOM_BUILD', value: "$BUILD_NUMBER"]]
  }
}

throttle(['loom-sdk']) {
  parallel builders
}
