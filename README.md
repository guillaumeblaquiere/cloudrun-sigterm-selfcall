# Overview

This repository demonstrates how to manage the [graceful termination with Cloud Run in Go](https://cloud.google.com/blog/topics/developers-practitioners/graceful-shutdowns-cloud-run-deep-dive) 
and how to call the service to always keep one instance warm and thus to minimize the cold starts.

This code illustrates [this article](https://medium.com/google-cloud/3-solutions-to-mitigate-the-cold-starts-on-cloud-run-8c60f0ae7894)

# Deployment

## Build

To build the container, you have several solution

**Cloud Build with Buildpack (Dockerfile is useless, you can delete it in this case)**

```
# Update with your project ID
PROJECT_ID=<MY_PROKECT_ID>

cloud builds submit --pack image=gcr.io/${PROJECT_ID}/self-call-sigterm
```

**Cloud Build with Dockerfile**

```
# Update with your project ID
PROJECT_ID=<MY_PROKECT_ID>

gcloud builds submit -t gcr.io/${PROJECT_ID}/self-call-sigterm
```
**Locally with docker**

```
# Update with your project ID
PROJECT_ID=<MY_PROKECT_ID>

docker build -t gcr.io/${PROJECT_ID}/self-call-sigterm .
docker push gcr.io/${PROJECT_ID}/self-call-sigterm

```

## Deployment

Simply deploy the service on Cloud Run

```
# Update with your project ID
PROJECT_ID=<MY_PROKECT_ID>

gcloud run deploy --platform=managed --region=us-central1 --image=gcr.io/${PROJECT_ID}/self-call-sigterm \
--allow-unauthenticated self-call-sigterm
```


And then call it to initialize.

```
curl $(gcloud run services describe --format="value(status.url)" self-call-sigterm)
```

# License

This library is licensed under Apache 2.0. Full license text is available in
[LICENSE](https://github.com/guillaumeblaquiere/cloudrun-sigterm-selfcall/tree/master/LICENSE).