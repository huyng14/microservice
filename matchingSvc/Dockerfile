FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

# Copy requirements first to leverage Docker layer caching
COPY requirements.txt /app/requirements.txt

# Install build deps, install Python packages without cache, then remove build deps and apt lists
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential gcc libffi-dev \
    && pip install --no-cache-dir --upgrade pip setuptools wheel \
    && pip install --no-cache-dir -r /app/requirements.txt \
    && apt-get purge -y --auto-remove build-essential gcc libffi-dev \
    && rm -rf /var/lib/apt/lists/* /root/.cache/pip

# Copy application source (after installing deps to keep cacheable layer)
COPY . /app

EXPOSE 9020

CMD ["python", "main.py"]
