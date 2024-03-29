FROM python:3.7.3-slim

LABEL source="https://github.com/eerkunt/terraform-compliance"

ENV TERRAFORM_VERSION=0.12.5

RUN  apt-get update && \
     apt-get install -y git curl unzip && \
     curl https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip --output terraform_linux_amd64.zip && \
     unzip terraform_linux_amd64.zip -d /usr/bin && \
     pip install terraform-compliance && \
     pip uninstall -y radish radish-bdd && \
     pip install radish radish-bdd && \
     rm -rf /var/lib/apt/lists/* && \
     mkdir -p /app 
     
#Adding API binary to app folder
ADD api/api /app
ADD api/example.tf /app
WORKDIR /app
RUN terraform init
CMD ["/app/api"]

