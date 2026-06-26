import logging
from pymongo import ReturnDocument
try:
    from pymongo import MongoClient
except ImportError:
    MongoClient = None

mongo_uri="mongodb+srv://skylab:skylab@consultatantaimatch.ftecqos.mongodb.net/"

# Configure logging with line numbers
logging.basicConfig(
    level=logging.DEBUG,
    format='[DEBUG] %(filename)s:%(lineno)d - %(message)s'
)
logging.disable(logging.DEBUG)  # Disable all DEBUG logs
logger = logging.getLogger(__name__)

def insert_to_mongo(data, db_name, collection_name):
    if MongoClient is None:
        logger.error("pymongo is not installed. Please install it with 'pip install pymongo'.")
        return None
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]
        result = collection.insert_one(data)
        logger.info(f"Inserted document with _id: {result.inserted_id}")
        return result.inserted_id
    except Exception as e:
        logger.error(f"Error inserting into MongoDB: {e}")
        return None
    
def update_to_mongo(data, db_name, collection_name):
    if MongoClient is None:
        logger.error("pymongo is not installed. Please install it with 'pip install pymongo'.")
        return None
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]
        # result = collection.update_one({"taskId": data["taskId"]}, {"$set": data})
        # logger.info(f"Matched: {result.matched_count}, Modified: {result.modified_count}")
        document = collection.find_one_and_update(
            {"taskId": data["taskId"]},
            {"$set": data},
            upsert=False,
            return_document=ReturnDocument.AFTER
        )        
        logger.info(f"Updated document _id: {document['_id']}")

        return document["_id"]
    except Exception as e:
        logger.error(f"Error updating MongoDB: {e}")
        return None
    
def find_uploaded_CV_by_taskId(taskId, db_name, collection_name):
    if MongoClient is None:
        logger.error("pymongo is not installed. Please install it with 'pip install pymongo'.")
        return None
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]
        document = collection.find_one({"taskId": taskId})
        if document:
            logger.info(f"Found document with taskId: {taskId}")
            return document
        else:
            logger.info(f"No document found with taskId: {taskId}")
            return None
    except Exception as e:
        logger.error(f"Error finding document in MongoDB: {e}")
        return None