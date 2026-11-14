FROM docker.io/dagster/dagster-celery-k8s:1.13.26

WORKDIR /opt/dagster/app
COPY requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
COPY signalops_dagster ./signalops_dagster
ENV PYTHONPATH=/opt/dagster/app
