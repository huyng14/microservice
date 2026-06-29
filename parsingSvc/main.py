import logging
import os
import re

from fastapi import FastAPI, UploadFile, File, Request
import threading
from fastapi.middleware.cors import CORSMiddleware
from dotenv import load_dotenv
from pathlib import Path
import uuid

from fastapi.staticfiles import StaticFiles

from parserWGemini import parse_resume_docx
from mongoDB.CV import find_uploaded_CV_by_taskId

app = FastAPI()

# Load environment variables from .env (if present)
load_dotenv()

# Read CORS origins from CORS_ALLOW_ORIGINS env var. Accept comma/space/semicolon-separated list.
cors_env = os.getenv("CORS_ALLOW_ORIGINS", "")
if cors_env:
    origins = [o.strip() for o in re.split(r"[,;\s]+", cors_env) if o.strip()]
else:
    # Fallback to previous hard-coded origins
    origins = [
        "http://localhost:5173",
    ]

# Log computed CORS origins for debugging
logger = logging.getLogger(__name__)
logger.info(f"CORS origins configured: {origins}")

# Enable CORS only for specified origins
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["POST", "GET"],
    allow_headers=["*"],
)
# Configure logging with line numbers
logging.basicConfig(
    level=logging.DEBUG,
    format='[DEBUG] %(filename)s:%(lineno)d - %(message)s'
)
logging.disable(logging.DEBUG)  # Disable all DEBUG logs
logger = logging.getLogger(__name__)

# Health check route
@app.get('/')
async def health_check():
    return {"status": "ok"}

# Create upload directory if it doesn't exist
UPLOAD_DIR = Path("uploads")
UPLOAD_DIR.mkdir(exist_ok=True)
# Serve uploaded files at /files
app.mount("/files", StaticFiles(directory=str(UPLOAD_DIR)), name="files")
    
@app.post("/upload/cv")
async def upload(request: Request, file: UploadFile = File(...)):
    # Generate a random taskId like CV_c07bfb93-1c5f-4d7d-9c39-26f3a0f79b18
    taskId = f"CV_{uuid.uuid4()}"
    file_path = UPLOAD_DIR / f"file-{taskId}.{file.filename.split('.')[-1]}"

    # Save file to disk
    with open(file_path, "wb") as buffer:
        content = await file.read()
        buffer.write(content)
    # logger.info(f"File '{file.filename}' uploaded and saved to '{file_path}'. content: {content[:100]}... (truncated)")

    # Call Gemini API to parse information, then Store result in the database
  
    logger.info(f"Processing: {file_path}")
    # Run parser in background
    threading.Thread(target=parse_resume_docx, args=(str(file_path), taskId, request), daemon=True).start()
    logger.info(f"Started background thread to parse {file_path} with taskId {taskId}")

    return {
        "filename": file.filename,
        "taskId": taskId,
        "size": len(content),
        "status": "PROCESSING",
        "message": "File uploaded and stored successfully. Parsing in progress."
    }

@app.get("/cv/{taskId}/result")
async def get_result(taskId: str):
    # Placeholder implementation - replace with actual result retrieval logic
    # status: PENDING, PROCESSING, COMPLETED, FAILED
    # Query the database for the document with the given taskId and return its status and data.
    uploaded_cv = find_uploaded_CV_by_taskId(taskId, "project", "uploaded_files")
    if not uploaded_cv:
        return {"taskId": taskId, "status": "NOT_FOUND", "data": None}
    return {"taskId": uploaded_cv.get("taskId"), 
            "status": uploaded_cv.get("status"), 
            "data": uploaded_cv.get("data", None)}

if __name__ == '__main__':
    logger.info('Parsing Service running on http://0.0.0.0:9030')