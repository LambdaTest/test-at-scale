<p align="center">
  <img src="https://www.lambdatest.com/blog/wp-content/uploads/2020/08/LambdaTest-320-180.png" />
</p>
<h1 align="center">Test At Scale</h1>

![N|Solid](https://www.lambdatest.com/resources/images/TAS_banner.png)

<p align="center">
  <b>Test Smarter, Release Faster with test-at-scale.</b>
</p>

<p align="center">
  <a href="https://github.com/LambdaTest/test-at-scale/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-Apache%20License%202.0.-blue" /></a>
  <a href="https://github.com/LambdaTest/test-at-scale/blob/main/CONTRIBUTING.md"><img src="https://img.shields.io/badge/contributions-welcome-brightgreen?logo=github" /></a>
  <a href="#build"><img src="https://github.com/lambdatest/test-at-scale/actions/workflows/main.yml/badge.svg" /></a>
  <a href="#lint"><img src="https://github.com/lambdatest/test-at-scale/actions/workflows/golangci-lint.yml/badge.svg" /></a>
  <a href="#stale"><img src="https://github.com/lambdatest/test-at-scale/actions/workflows/stale.yml/badge.svg" /></a>
  <a href="https://discord.gg/Wyf8srhf6K"><img src="https://img.shields.io/badge/Discord-5865F2" /></a>

</p>

## Test at scale - TAS
TAS helps you accelerate your testing, shorten job times and get faster feedback on code changes, manage flaky tests and keep master green at all times.
<br/>

To learn more about TAS features and capabilities, see our [product page](https://www.lambdatest.com/test-at-scale). 

## Features
- Smart test selection to run only the subset of tests which get impacted by a commit ⚡
- Smart auto grouping of test to evenly distribute test execution across multiple containers based on previous execution times
- Deep insights about test runs and execution metrics
- Support status checks for pull requests
- Advanced analytics to surface test performance and quality data
- YAML driven declarative workflow management
- Natively integrates with Github and Gitlab
- Flexible workflow to run pre-merge and post-merge tests
- Allows blocking and unblocking tests directly from the UI or YAML directive. No more WIP commits!
- Support for customizing testing environment using raw commands in pre and poststeps
- Supports Javascript monorepos
- Smart depdency caching to speedup subsequent test runs
- Easily customizable to support all major language and frameworks
- Available as (https://lambdatest.com/test-at-scale)[hosted solution] as well as self-hosted opensource runner
- [Upcoming] Smart flaky test management 🪄

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

  ![N|Solid](https://www.lambdatest.com/support/assets/images/signup_gi-f776f9b5a6ad4d877e6c427094969e1e.gif)
  
- Select **TAS Self Hosted** and click on Proceed.
- You will find your **LambdaTest Secret Key** on this page which will be required in the next steps.

  ![N|Solid](https://www.lambdatest.com/support/assets/images/synapse-e3e691d8f3d08ff6b3b2ced1a9ef61ed.gif)

<br>

### Step 2 - Creating a configuration file for self hosted setup

Before installation we need to create a file that will be used for configuring test-at-scale. 

- Open any `Terminal` of your choice.
- Move to your desired directory or you can create a new directory and move to it using the following command.
- Download our sample configuration file using the given command.

```bash
mkdir ~/test-at-scale
cd ~/test-at-scale
curl https://raw.githubusercontent.com/LambdaTest/test-at-scale/main/.sample.synapse.json -o .synapse.json
```
- Open the downloaded `.synapse.json` configuration file in any editor of your choice.
- You will need to add the following in this file: 
  - 1- **LambdaTest Secret Key**, that you got at the end of **Step 1**.
  - 2- **Git Token**, that would be required to clone the repositories after Step 3. Generating [GitHub](https://www.lambdatest.com/support/docs/tas-how-to-guides-gh-token), [GitLab](https://www.lambdatest.com/support/docs/tas-how-to-guides-gl-token) personal access token.
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
  cd ~/test-at-scale
  curl -L https://raw.githubusercontent.com/LambdaTest/test-at-scale/main/docker-compose.yml -o docker-compose.yml
  docker-compose up -d
  ```

> **NOTE:** This docker-compose file will pull the latest version of test-at-scale and install on your self hosted environment.

<details id="docker">
<summary>Installation without <b>Docker Compose</b></summary>

To get up and running quickly, you can use the following instructions to setup Test at Scale on Self hosted environment without docker-compose.

- The `.synapse.json` configuration file made in [Step 2](#step-2---creating-a-configuration-file-for-self-hosted-setup) will be required before executing the next command.
- Execute the following command to run Test at Scale docker container

```bash
cd ~/test-at-scale
docker network create --internal test-at-scale
docker run --name synapse --restart always \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /tmp/synapse:/tmp/synapse \
    -v ${PWD}/.synapse.json:/home/synapse/.synapse.json \
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

  ![N|Solid](https://www.lambdatest.com/support/assets/images/import-postmerge-c1b26a9e78a1b63dc23dd2129b16f9d6.gif)

<br>

### Step 5 - Configuring TAS yml
A `.tas.yml` file is a basic yaml configuration file that contains steps required for installing necessary dependencies and executing the tests present in your repository.
- In order to configure your imported repository, follow the steps given on the `.tas.yml`  configuration page. 
- You can also know more about `.tas.yml` configuration parameters [here](https://www.lambdatest.com/support/docs/tas-configuring-tas-yml).

  ![N|Solid](https://www.lambdatest.com/support/assets/images/yml-download-6e7366b290de36ef8cb74f3d564850af.gif)
  
- Placing the `.tas.yml` configuration file.
  - Create a new file as **.tas.yml** at the root level of your repository .
  - **Copy** the configuration from the TAS yml configuration page and **paste** them in the **.tas.yml** file you just created.
  - **Commit and Push** the changes to your repo.   
  
  ![N|Solid](https://www.lambdatest.com/support/assets/images/yml_placing-72cd952b403e499a938151c955540e18.gif)

## **Language & Framework Support** 
Currently we support Mocha, Jest and Jasmine for testing Javascript codebases.

## **Tutorials**
- [Setting up you first repo on TAS - Cloud](https://www.lambdatest.com/support/docs/tas-getting-started-integrating-your-first-repo/) 
- [Setting up you first repo on TAS - Self Hosted](https://www.lambdatest.com/support/docs/tas-self-hosted-installation) 
- Sample repos : [Mocha](https://github.com/LambdaTest/mocha-demos), [Jest](https://github.com/LambdaTest/jest-demos), [Jasmine](https://github.com/LambdaTest/jasmine-node-js-example).
- [How to configure a .tas.yml file](https://www.lambdatest.com/support/docs/tas-configuring-tas-yml)

## **Contribute**
We love our contributors! If you'd like to contribute anything from a bug fix to a feature update, start here:

- 📕 Read our Code of Conduct [Code of Conduct](/CODE_OF_CONDUCT.md).
- 📖 Know more about [test-at-scale](/CONTRIBUTING.md#repo-overview) and contributing from our [Contribution Guide](/CONTRIBUTING.md).
- 👾 Explore some good first issues [good first issues](https://github.com/LambdaTest/test-at-scale/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22).

### **Join our community**
Engage with Developers, SDETs, and Testers around the world. 
- Get the latest product updates. 
- Discuss testing philosophies and more. 
Join the Test-at-scale Community on [Discord](https://discord.gg/Wyf8srhf6K). Click [here](https://discord.com/channels/940635450509504523/941297958954102846) if you are already an existing member.

### **Support & Troubleshooting** 
The documentation and community will help you troubleshoot most issues. If you have encountered a bug, you can contact us using one of the following channels:
- Help yourself with our [Documentation](https://www.lambdatest.com/support/docs/tas-overview)📚, and [FAQs](https://www.lambdatest.com/support/docs/tas-faq/).
- In case of Issue & bugs go to [GitHub issues](https://github.com/LambdaTest/test-at-scale/issues)🐛.
- For support & feedback join our [Discord](https://discord.gg/Wyf8srhf6K) or reach out to us on our [email](mailto:hello.tas@lambdatest.com)💬. 

We are committed to fostering an open and welcoming environment in the community. Please see the Code of Conduct.

## **License**

TestAtScale is available under the [Apache License 2.0](https://github.com/LambdaTest/test-at-scale/blob/main/LICENSE). Use it wisely.
