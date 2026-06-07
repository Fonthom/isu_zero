FROM python:3.11-slim AS base

# System dependencies needed by OpenCV and psycopg2
RUN apt-get update && apt-get install -y --no-install-recommends \
    libglib2.0-0 \
    libgl1 \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install Python dependencies first — cached unless requirements.txt changes
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Download YOLOv8 nano model weights at build time
# so the container never needs internet access at runtime
RUN python -c "from ultralytics import YOLO; YOLO('yolov8n.pt')"

# Copy source
COPY . .

CMD ["python", "main.py"]