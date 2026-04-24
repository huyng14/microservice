from google import genai
from google.genai import types
import numpy as np
import logging
import os
from dotenv import load_dotenv

from pymongo import MongoClient
from bson.binary import Binary

# Configure logging with line numbers
logging.basicConfig(
    level=logging.DEBUG,
    format='[DEBUG] %(filename)s:%(lineno)d - %(message)s'
)
logger = logging.getLogger(__name__)

# Load environment variables from .env file
load_dotenv()

GENAI_API_KEY = os.environ.get("GENAI_API_KEY")
MONGO_URI = os.environ.get("MONGO_URI")

client = genai.Client(api_key=GENAI_API_KEY)
mongo_uri = MONGO_URI

def read_from_database(db_name, collection_name):
    # mongo_uri = "mongodb+srv://skylab:skylab@consultatantaimatch.ftecqos.mongodb.net/"
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]

        # Read all documents into a list
        documents = list(collection.find())

        return documents

    except Exception as e:
        logger.error(f"Error reading from MongoDB: {e}")
        return None

def update_embedding_in_database(db_name, collection_name, document_id, input_data, field_name="embedding"):
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]
        
        # Convert list of floats to float32 binary
        embedding_array = np.array(input_data, dtype=np.float32)
        binary_data = Binary(embedding_array.tobytes())
        # Update the document with the new embedding vector
        result = collection.update_one(
            {"_id": document_id},
            {"$set": {field_name: binary_data}}
        )

        if result.modified_count > 0:
            logger.info(f"Successfully updated document with _id: {document_id}")
        else:
            logger.info(f"No document found with _id: {document_id} or no changes made.")

    except Exception as e:
        logger.error(f"Error updating MongoDB: {e}")

def work_exp_embedding_models(database_name="project", collection_name="consultants"):
    resume = read_from_database(database_name, collection_name)
    for doc in resume:
        logger.info(f"doc _id is {doc['_id']}")
        logger.info(f"doc Name is {doc['name']}")
        logger.info(f"type of doc Work Experience is {type(doc['experience'])} - is None: {doc['experience'] is None}")
        
        if "experience_embedding" not in doc and doc["experience"] is not None and len(doc["experience"]) > 0:
        # work_experience_embedding does not exist, generating...
            result = client.models.embed_content(
                model="gemini-embedding-001",
                contents=str(doc["experience"]))
                # config=types.EmbedContentConfig(task_type="SEMANTIC_SIMILARITY")).embeddings

            logger.debug(f"result.embeddings type is {type(result.embeddings)}, len = {len(result.embeddings)}")
            # print(result.embeddings.values)
            for e in result.embeddings:
                logger.debug(f"e values type is {type(e.values)}, len = {len(e.values)}")
                update_embedding_in_database(
                database_name,
                collection_name,
                doc["_id"],
                e.values,
                field_name="experience_embedding")
        else:
            # experience_embedding already exists, skipping...
            continue
    return

def assignment_desc_embedding_models(database_name="project", collection_name="assignments"):
    assignments = read_from_database(database_name, collection_name)
    for doc in assignments:
        logger.info(f"doc _id is {doc['_id']}")
        logger.info(f"type of doc Description is {type(doc['description'])} - is None: {doc['description'] is None}")
        
        if "description_embedding" not in doc and doc["description"] is not None and len(doc["description"]) > 0:
            # description_embedding does not exist, generating...
            result = client.models.embed_content(
                model="gemini-embedding-001",
                contents=str(doc["description"]))
                # config=types.EmbedContentConfig(task_type="SEMANTIC_SIMILARITY")).embeddings

            logger.info(f"result.embeddings type is {type(result.embeddings)}, len = {len(result.embeddings)}")
            for e in result.embeddings:
                logger.info(f"e values type is {type(e.values)}, len = {len(e.values)}")
                update_embedding_in_database(
                    database_name,
                    collection_name,
                    doc["_id"],
                    e.values,
                    field_name="description_embedding")
        else:
            # description_embedding already exists, skipping...
            continue
    return

if __name__ == "__main__":
    # Standalone usage examples
    
    # Generate embeddings for all consultants
    # work_exp_embedding_models()
    
    # Generate embeddings for all assignments
    # assignment_desc_embedding_models()
    
    print("Use main.py for orchestrated pipeline execution.")

