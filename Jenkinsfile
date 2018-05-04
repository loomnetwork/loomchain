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

builders['linux'] = {
  node('linux') {
    try {
      stage ('Checkout - Linux') {
        checkout changelog: true, poll: true, scm:
        [
          $class: 'GitSCM',
          branches: [[name: '**']],
          doGenerateSubmoduleConfigurations: false,
          extensions: [
            [$class: 'PreBuildMerge',
            options: [
              fastForwardMode: 'FF',
              mergeRemote: 'origin',
              mergeTarget: 'master'
              ]
            ]],
          submoduleCfg: [],
          userRemoteConfigs:
          [[
            credentialsId: 'loom-sdk',
            url: 'git@github.com:loomnetwork/loomchain.git']
          ]]
      }

      setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Linux");

      stage ('Build - Linux') {
        sh '''
          # For local merge
          git config user.email "jenkins@loomx.io"
          git config user.name "Jenkins"
          ./jenkins.sh
          cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
          gsutil cp loom gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loom
        '''
      }
    } catch (e) {
      throw e
    } finally {
      if (currentBuild.currentResult == 'FAILURE') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Linux");
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Linux");
      }
    }
  }
}

builders['windows'] = {
  node('windows') {
    try {
      stage ('Checkout - Windows') {
        checkout changelog: true, poll: true, scm:
        [
          $class: 'GitSCM',
          branches: [[name: '**']],
          doGenerateSubmoduleConfigurations: false,
          extensions: [
            [$class: 'PreBuildMerge',
            options: [
              fastForwardMode: 'FF',
              mergeRemote: 'origin',
              mergeTarget: 'master'
              ]
            ]],
          submoduleCfg: [],
          userRemoteConfigs:
          [[
            credentialsId: 'loom-sdk',
            url: 'git@github.com:loomnetwork/loomchain.git']
          ]]
      }

      setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Windows");

      stage ('Build - Windows') {
        bat '''
          jenkins.cmd
          SET PATH=C:\\Program Files (x86)\\Google\\Cloud SDK\\google-cloud-sdk\\bin;%PATH%;
          cd \\msys64\\tmp\\gopath-${BUILD_TAG}\\src\\github.com\\loomnetwork\\loomchain
          gsutil cp loom.exe gs://private.delegatecall.com/loom/windows/build-$BUILD_NUMBER/loom.exe
        '''
      }
    } catch (e) {
      throw e
    } finally {
      if (currentBuild.currentResult == 'FAILURE') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Windows");
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Windows");
      }
    }
  }
}

builders['osx'] = {
  node('osx') {
    try {
      stage ('Checkout - OSX') {
        checkout changelog: true, poll: true, scm:
        [
          $class: 'GitSCM',
          branches: [[name: '**']],
          doGenerateSubmoduleConfigurations: false,
          extensions: [
            [$class: 'PreBuildMerge',
            options: [
              fastForwardMode: 'FF',
              mergeRemote: 'origin',
              mergeTarget: 'master'
              ]
            ]],
          submoduleCfg: [],
          userRemoteConfigs:
          [[
            credentialsId: 'loom-sdk',
            url: 'git@github.com:loomnetwork/loomchain.git']
          ]]
      }

      setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "OSX");

      stage ('Build - OSX') {
        sh '''
          ./jenkins.sh
          cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
          gsutil cp loom gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/loom
        '''
      }
    } catch (e) {
      throw e
    } finally {
      if (currentBuild.currentResult == 'FAILURE') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "OSX");
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "OSX");
      }
    }
  }
}

parallel builders
