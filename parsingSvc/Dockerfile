# Use an official Python runtime as a parent image
FROM python:3.12-slim

# Set working directory
WORKDIR /app

# Install system dependencies needed for Python packages and runtime
RUN apt-get update && \
    apt-get install -y --no-install-recommends gcc libffi-dev libpq-dev && \
    rm -rf /var/lib/apt/lists/*

# Copy requirements and install them
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# Copy application source
COPY . /app

# Create uploads directory and ensure it exists at runtime
RUN mkdir -p /app/uploads

# Expose the port used by the FastAPI application
EXPOSE 9030

# Use a non-root user for better container security
RUN useradd --create-home appuser && chown -R appuser /app
USER appuser

# Start the FastAPI app using Python module invocation of Uvicorn
CMD ["python", "-m", "uvicorn", "main:app", "--reload", "--host", "0.0.0.0", "--port", "9030"]
