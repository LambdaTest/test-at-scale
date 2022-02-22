<p align="center">
  <img src="https://www.lambdatest.com/blog/wp-content/uploads/2020/08/LambdaTest-320-180.png" />
</p>
<h1 align="center">Test At Scale</h1>

![N|Solid](https://www.lambdatest.com/resources/images/TAS_banner.png)

<p align="center">
  <b>Test Smarter, Release Faster with test-at-scale.</b>
</p>

<p align="center">
  <a href="https://github.com/LambdaTest/test-at-scale/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-Apache%20License%202.0.-blue" /></a>
  <a href="#build"><img src="https://github.com/lambdatest/test-at-scale/actions/workflows/main.yml/badge.svg" /></a>
  <a href="#lint"><img src="https://github.com/lambdatest/test-at-scale/actions/workflows/golangci-lint.yml/badge.svg" /></a>
  <a href="#stale"><img src="https://github.com/lambdatest/test-at-scale/actions/workflows/stale.yml/badge.svg" /></a>
  <a href="https://discord.gg/Wyf8srhf6K"><img src="https://img.shields.io/badge/Discord-5865F2" /></a>

</p>

## Table of contents 
- 🚀 [Getting Started](#getting-started)
- 💡 [Tutorials](#tutorials)
- 💖 [Contribute](#contribute)
- 📖 [Docs](https://www.lambdatest.com/support/docs/tas-overview)

## Getting Started

### Step 1 - Setting up a New Account
In order to create an account, visit [TAS Login Page](https://tas.lambdatest.com/login/). (Or [TAS Home Page](https://tas.lambdatest.com/))
- Login using a suitable git provider and select your organization you want to continue with.
- Tell us your specialization, team size. 

  ![N|Solid](https://www.lambdatest.com/support/assets/images/signup_gi-c46290845329881e7893705add21d7cd.gif)
  
- Select **TAS Self Hosted** and click on Proceed.
- You will find your **LambdaTest Secret Key** on this page which will be required in the next steps.

  ![N|Solid](https://www.lambdatest.com/support/assets/images/synapse-b3e8b6b475967d82bbee0d56339daf5a.gif)

<br>

### Step 2 - Creating a configuration file for self hosted setup

Before installation we need to create a file that will be used for configuring test-at-scale. 

- Open any `Terminal` of your choice.
- Move to your desired directory or you can create a new directory and move to it using the following command.
- Download our sample configuration file using the given command.

```bash
mkdir ~/test-at-scale
cd ~/test-at-scale
curl https://raw.githubusercontent.com/LambdaTest/test-at-scale/master/.sample.synapse.json -o .synapse.json
```
- Open the downloaded `.synapse.json` configuration file in any editor of your choice.
- You will need to add the following in this file: 
  - 1- **LambdaTest Secret Key**, that you got at the end of **Step 1**.
  - 2- **Git Token**, that would be required to clone the repositories after Step 3. Generating[GitHub](https://www.lambdatest.com/support/docs/tas-how-to-guides-gh-token), [GitLab](https://www.lambdatest.com/support/docs/tas-how-to-guides-gl-token) personal access token.
- This file will also be used to store certain other parameters such as **Repository Secrets** (Optional), **Container Registry** (Optional) etc that might be required in configuring test-at-scale on your local/self-hosted environment. You can learn more about the configuration options [here](https://www.lambdatest.com/support/docs/tas-self-hosted-configuration#parameters).

<br>

### Step 3 - Installation

#### Installation on Docker

##### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) or [Docker-Compose](https://docs.docker.com/compose/install/) (Recommended)

##### Docker Compose
- Run the docker application.
  
  ```bash
  docker info --format "CPU: {{.NCPU}}, RAM: {{.MemTotal}}"
  ```
- Execute the above command to ensure that resources usable by Docker are atleast `CPU: 2, RAM: 4294967296`.
  > **NOTE:** In order to run test-at-scale you require a minimum configuration of 2 CPU cores and 4 GBs of RAM.

- The `.synapse.json` configuration file made in [Step 2](#step-2---creating-a-configuration-file-for-self-hosted-setup) will be required before executing the next command.
- Download and run the docker compose file using the following command.
  
  ```bash
  curl -L https://raw.githubusercontent.com/LambdaTest/test-at-scale/master/docker-compose.yml -o docker-compose.yml
  docker-compose up -d
  ```

> **NOTE:** This docker-compose file will pull the latest version of test-at-scale and install on your self hosted environment.

<details id="docker">
<summary>Installation without <b>Docker Compose</b></summary>

To get up and running quickly, you can use the following instructions to setup Test at Scale on Self hosted environment without docker-compose.

- The `.synapse.json` configuration file made in [Step 2](#step-2---creating-a-configuration-file-for-self-hosted-setup) will be required before executing the next command.
- Execute the following command to run Test at Scale docker container

```bash
docker network create --internal test-at-scale
docker run —name synapse —-restart always \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /tmp/synapse:/tmp/synapse \
    -v .synapse.json:/home/synapse/.synapse.json \
    -v /etc/machine-id:/etc/machine-id \
    --network=test-at-scale \
    lambdatest/synapse:latest
```
> **WARNING:** We strongly recommend to use docker-compose while Test at Scale on Self hosted environment.

</details>  

<details>
<summary>Installation on <b> Local Machine </b> & <b> Supported Cloud Platforms </b> </summary>

- Local Machine - Setup using [docker](#docker).
- Setup on [Azure](https://www.lambdatest.com/support/docs/tas-self-hosted-installation#azure)
- Setup on [AWS](https://www.lambdatest.com/support/docs/tas-self-hosted-installation#aws)
- Setup on [GCP](https://www.lambdatest.com/support/docs/tas-self-hosted-installation#gcp)
</details>

- Once the installation is complete, go back to the TAS portal.
- Click the 'Test Connection' button to ensure `test-at-scale` self hosted environment is connected and ready.
- Hit `Proceed` to move forward to [Step 4](#step-4---importing-your-repo)


<br>

### Step 4 - Importing your repo
> **NOTE:** Currently we support Mocha, Jest and Jasmine for testing Javascript codebases.
- Click the Import button for the `JS` repository you want to integrate with TAS.
- Once Imported successfully, click on `Go to Project` to proceed further.
- You will be asked to setup a `post-merge` here. We recommend to proceed ahead with default settings. (You can change these later.) 

  ![N|Solid](https://www.lambdatest.com/support/assets/images/import-postmerge-b6f7146b6b43d5f8876ec9bb73a478a1.gif)

### Step 5 - Configuring TAS yml
A `.tas.yml` file is a basic yaml configuration file that contains steps required for installing necessary dependencies and executing the tests present in your repository.
- In order to configure your imported repository, follow the steps given on the `.tas.yml`  configuration page. 
- You can also know more about `.tas.yml` configuration parameters [here](https://www.lambdatest.com/support/docs/tas-configuring-tas-yml).

  ![N|Solid](https://www.lambdatest.com/support/assets/images/yml-download-375c25fabbe3fe533782b94adecd2f95.gif)

## **Language & Framework Support** 
Currently we support Mocha, Jest and Jasmine for testing Javascript codebases.

## **Tutorials**
- [Setting up you first repo on TAS - Cloud](https://www.lambdatest.com/support/docs/tas-getting-started-integrating-your-first-repo/) (Sample repos : [Mocha](https://github.com/LambdaTest/mocha-demos), [Jest](https://github.com/LambdaTest/jest-demos), [Jasmine](https://github.com/LambdaTest/jasmine-node-js-example).)
- [Setting up you first repo on TAS - Self Hosted](https://www.lambdatest.com/support/docs/tas-self-hosted-installation) (Sample repos : [Mocha](https://github.com/LambdaTest/mocha-demos), [Jest](https://github.com/LambdaTest/jest-demos), [Jasmine](https://github.com/LambdaTest/jasmine-node-js-example).)
- [How to configure a .tas.yml file](https://www.lambdatest.com/support/docs/tas-configuring-tas-yml)

## **Contribute**
We love our contributors! If you'd like to contribute anything from a bug fix to a feature update, start here:

- 📕 Read our Code of Conduct [Code of Conduct](/CODE_OF_CONDUCT.md).
- 📖 Know more about [test-at-scale](/CONTRIBUTING.md#repo-overview) and contributing from our [Contribution Guide](/CONTRIBUTING.md).
- 👾 Explore some good first issues [good first issues](https://github.com/LambdaTest/test-at-scale/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22).

### **Join our community**
Engage with Developers, SDETs, and Testers around the world. Get the latest product updates. Discuss testing philosophies and more. Join the Test-at-scale Community on [Discord](https://discord.gg/Wyf8srhf6K).

### **Support & Troubleshooting** 
The documentation and community will help you troubleshoot most issues. If you have encountered a bug, you can contact us using one of the following channels:
- Help yourself with our [Documentation](https://www.lambdatest.com/support/docs/tas-overview)📚.
- In case of Issue & bugs go to [GitHub issues](https://github.com/LambdaTest/test-at-scale/issues)🐛.
- For support & feedback join our [Discord](https://discord.gg/Wyf8srhf6K) or reach out to us on our [email](mailto:hello.tas@lambdatest.com)💬. 

We are committed to fostering an open and welcoming environment in the community. Please see the Code of Conduct.

## **License**

TestAtScale is available under the [Apache License 2.0](https://github.com/LambdaTest/test-at-scale/blob/master/LICENSE). Use it wisely.
