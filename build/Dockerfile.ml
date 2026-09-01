FROM python:3.12-slim

WORKDIR /app

COPY ml/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Full ml/ tree (app code + trained artifacts). .dockerignore excludes
# .venv / __pycache__ / heavy data.
COPY ml/ /app/ml/

ENV PYTHONUNBUFFERED=1
EXPOSE 5001

# python -m ml.run is the supported entrypoint (repo root on sys.path).
CMD ["python", "-m", "ml.run"]
