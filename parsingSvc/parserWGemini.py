import os

from dotenv import load_dotenv
from google import genai
from google.genai import errors
import docx2txt
import json

from mongoDB.CV import insert_to_mongo, update_to_mongo
import logging

# Load environment variables from .env (if present)
load_dotenv()

# Read GENAI_API_KEY from environment variable
genai_api_key = os.getenv("GENAI_API_KEY", "")
client = genai.Client(api_key=genai_api_key)

# Configure logging with line numbers
logging.basicConfig(
    level=logging.DEBUG,
    format='[DEBUG] %(filename)s:%(lineno)d - %(message)s'
)
logging.disable(logging.DEBUG)  # Disable all DEBUG logs
logger = logging.getLogger(__name__)

def parse_resume_docx(file_path, taskId=None):
    # Ensure a taskId is set (generate if not provided)
    if not taskId:
        return None
    
    raw = docx2txt.process(file_path)
    text = "\n".join([ln.strip() for ln in raw.splitlines() if ln.strip()])

    insert_to_mongo({"taskId": taskId, "status": "PROCESSING"}, 
                    "project", "uploaded_files")

    prompt = f"""
    Extract data from the résumé into a JSON object matching this TypeScript schema.

    Schema:
    interface CV {{ id: string; name: string; email: string; phone: string; title: string; summary: string; skills: string[]; experience: Experience[]; education: Education[]; }}
    interface Experience {{ id: string; company: string; position: string; startDate: string; endDate: string; description: string; }}
    interface Education {{ id: string; institution: string; degree: string; field: string; graduationDate: string; }}

    Résumé:
    {text}

    JSON:
    """

    # Call Gemini API
    
    response = None
   
    try:
        response = client.models.generate_content(
            model="gemini-2.5-flash",
            contents=prompt
        )
    except errors.ServerError as e:
        logger.error(f"Server error for task {taskId}; marking as FAILED")
        return update_to_mongo({"data": str(e), "taskId": taskId, "status": "FAILED"}, 
                                "project", "uploaded_files")
    except errors.APIError as e:
        logger.error(f"API error calling Gemini: {e}")
        return update_to_mongo({"data": str(e), "taskId": taskId, "status": "FAILED"}, 
                                "project", "uploaded_files")
    except Exception as e:
        logger.error(f"Unexpected error calling Gemini API: {e}")
        return update_to_mongo({"data": str(e), "taskId": taskId, "status": "FAILED"}, 
                                "project", "uploaded_files")
   
    # Parse response.text as JSON
    try:
        lines = response.text.splitlines()
        cleaned_text = "\n".join(lines[1:-1])
        # print("Cleaned Text:\n", cleaned_text)
        resume_data = json.loads(cleaned_text)
    except Exception as e:
        logger.error(f"Error parsing response as JSON: {e}")
        return update_to_mongo({"data": str(e), "taskId": taskId, "status": "FAILED"}, 
                           "project", "uploaded_files")

    # Insert resume_data using the helper
    return update_to_mongo({"data": resume_data, "taskId": taskId, "status": "COMPLETED"}, 
                           "project", "uploaded_files")
