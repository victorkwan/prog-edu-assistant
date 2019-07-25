# Google Compute Engine (GCE)

## Initial gcloud authentication (on a dev machine)

You need to install recent version of Google Cloud SDK first.

    # Authenticate with gcloud 
    gloud auth login
    # Choose the project name
    gcloud config set project prog-edu-assistant

## Build and push images to GCR (on a dev machine)

You only need to run this step if you have made changes to the source code base.

    (cd docker && ./build.sh && \
     docker tag server asia.gcr.io/prog-edu-assistant/server && \
     docker tag worker asia.gcr.io/prog-edu-assistant/worker && \
     docker push asia.gcr.io/prog-edu-assistant/server && \
     docker push asia.gcr.io/prog-edu-assistant/worker)

## Create an instance (on a dev machine)

    gcloud compute instances create prog-edu-assistant \
      --zone=asia-northeast1-b \
      --machine-type=n1-standard-1 \
      --image-family=cos-stable \
      --image-project=cos-cloud \
      --tags=http-server,https-server

    # See the IP address of the instance:
    gcloud compute instances list

Note: secret.env has two items that depend on the stable server address:

(1) `SERVER_URL` should contain the URL of the server, starting with http://
and having the port, but without the final slash. Obviously the stable URL
of the server should resolve to the actual IP address of the instance.
Perhaps it is a good idea to configure instance with a static IP address.
Below I am using `prog-edu-assistant.salikh.info` as a stable name, but
you will need to choose a name that you can control and update.

(2) The `CLIENT_ID` and `CLIENT_SECRET` used for OpenID Connect authentication
must list the domain of the server as an authorized domain, as well
as have the URL http://server:port/upload in the authorized redirect URI list.

The file `service-account.json` should be obtained from GCP console as a
service account key.

    # Copy the deployment files to the instance:
    scp -r deploy/{certs,docker-compose.yml,secret.env,service-account.json} \
      prog-edu-assistant.salikh.info:

    
## Start the autochecker server (on a GCE instance)

Start with logging to console:

    ssh prog-edu-assistant.salikh.info
    mkdir -p logs
    docker pull asia.gcr.io/prog-edu-assistant/worker
    docker pull asia.gcr.io/prog-edu-assistant/server
    docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $PWD:$PWD -w=$PWD --entrypoint=sh docker/compose:1.24.0 -c 'cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io && docker-compose up --scale worker=4'

Or start and detach (on a dev machine):

    ssh prog-edu-assistant.salikh.info "mkdir -p logs && docker pull asia.gcr.io/prog-edu-assistant/worker && docker pull asia.gcr.io/prog-edu-assistant/server && docker run -d --rm -v /var/run/docker.sock:/var/run/docker.sock -v \$PWD:\$PWD -w=\$PWD --entrypoint=sh docker/compose:1.24.0 -c 'cat service-account.json | docker login -u _json_key --password-stdin https://asia.gcr.io && docker-compose up --scale worker=4'"

## Inspect running services on the GCE instance

    ssh prog-edu-assistant.salikh.info
    docker ps

## Kill all services (without taking the GCE instance down)

    ssh prog-edu-assistant.salikh.info
    docker ps -q | xargs -n1 docker kill

## Delete the instance after it is no longer needed (on a dev machine)

    gcloud compute instances delete prog-edu-assistant
