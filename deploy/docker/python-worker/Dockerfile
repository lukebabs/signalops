FROM python:3.12-slim AS python-worker

WORKDIR /app

COPY python/requirements-worker.txt ./python/requirements-worker.txt
RUN pip install --no-cache-dir -r ./python/requirements-worker.txt

COPY python ./python
COPY contracts ./contracts

ENV PYTHONPATH=/app/python

ENTRYPOINT ["python", "-m", "signalops_workers"]
