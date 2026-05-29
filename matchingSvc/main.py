import logging

from flask import Flask, jsonify, request
from flask_cors import CORS

from embeddingWGemini import ( work_exp_embedding_models, 
                                assignment_desc_embedding_models )
from comparing_vector import ( compare_1consultant_with_1assignment_and_explain, compare_consultant_with_assignments, 
                                find_ID_from_database, 
                                find_objectID_from_database,
                                generate_explanation_toConsultant, generate_explanation_toHiring_manager)

app = Flask(__name__)
# Enable CORS only for specified origins
CORS(app, resources={r"/*": {"origins": [
    "http://s3-demo-web-497559249788-ap-southeast-1-an.s3-website-ap-southeast-1.amazonaws.com",
    "http://localhost:5173"
]}})
# Configure logging with line numbers
logging.basicConfig(
    level=logging.DEBUG,
    format='[DEBUG] %(filename)s:%(lineno)d - %(message)s'
)
logging.disable(logging.DEBUG)  # Disable all DEBUG logs
logger = logging.getLogger(__name__)


def embed_phase(database_name="project", consultant_name="consultants", assignment_collection="assignments"):
    """
    Phase 2: Generate embeddings for parsed documents.
    
    Returns:
        dict with keys 'consultants_embedded', 'assignments_embedded', 'status'
    """
    print("\n" + "="*60)
    print("PHASE 2: GENERATING EMBEDDINGS")
    print("="*60)
    
    result = {
        "consultants_embedded": 0,
        "assignments_embedded": 0,
        "status": "pending"
    }
    
    try:
        print("\nGenerating embeddings for consultant work experience...")
        work_exp_embedding_models(database_name=database_name, collection_name=consultant_name)
        result["consultants_embedded"] += 1
        
        print("\nGenerating embeddings for assignment descriptions...")
        assignment_desc_embedding_models(database_name=database_name, collection_name=assignment_collection)
        result["assignments_embedded"] += 1
        
        result["status"] = "completed"
        logger.debug("\n✓ Embedding generation completed")
    except Exception as e:
        result["status"] = f"error: {str(e)}"
        logger.debug(f"\n✗ Embedding generation failed: {str(e)}")
    
    return result

    
def get_matched_score(consultant_id=None, assignment_id=None, explain=False, database="project", 
                  collection_consultants="consultants", collection_assignments="jobs"):
    if not consultant_id or not assignment_id:
        error_msg = "Error: Please provide both consultant_id and assignment_id"
        logger.error(error_msg)
        return {"error": error_msg, "response": None}
    try:
        logger.info(f"\nCalculating similarity score between consultant {consultant_id} and assignment {assignment_id}...")
        consultant = find_ID_from_database(database, collection_consultants, consultant_id)
        assignment = find_ID_from_database(database, collection_assignments, assignment_id)
        
        if not consultant:
            error_msg = f"Consultant {consultant_id} not found in database."
            logger.error(error_msg)
            return {"error": error_msg, "response": None}
        
        if not assignment:
            error_msg = f"Assignment {assignment_id} not found in database."
            logger.error(error_msg)
            return {"error": error_msg, "response": None}
        
        response = compare_1consultant_with_1assignment_and_explain(consultant_id, assignment_id, explain=explain, 
                                                                 database=database, 
                                                                 collection_consultants=collection_consultants, 
                                                                 collection_assignments=collection_assignments)
        # print(f"Similarity score: {response.get('score', 0):.4f}, Explanation: {response.get('explanation', 'N/A')}")
        response = {
            "consultant_id": str(consultant_id),
            "assignment_id": str(assignment_id),
            "score": response.get("score", 0),
            "explanation": response.get("explanation")
        }
        return {"error": None, "response": response}
    
    except Exception as e:
        error_msg = f"Exception in get_matched_score: {str(e)}"
        logger.error(error_msg)
        return {"error": error_msg, "response": None}

@app.route('/matching/<consultant_id>', methods=['POST'])
def matching(consultant_id):
    # time.sleep(5)  # Simulate processing time
    logger.info(f"Received matching request for consultant_id: {consultant_id}")
    
    # Get assignment_id from JSON request body
    data = request.get_json()
    assignment_id = data.get('assignment_id') if data else None
    
    compare_result = get_matched_score(consultant_id=consultant_id, assignment_id=assignment_id, explain=True
                                        , database="project", 
                                        collection_consultants="consultants", 
                                        collection_assignments="jobs")
    
    if compare_result['error']:
        return jsonify({
            'error': compare_result['error'],
            'response': None
        }), 400
    
    return jsonify({
        'error': None,
        'response': compare_result['response']
    }), 200


# Health check route
@app.route('/', methods=['GET'])
def health_check():
    return jsonify({"status": "ok"}), 200

if __name__ == '__main__':
    embed_result = embed_phase("project", "consultants", "jobs")
    if embed_result["status"] == "completed":
        logger.debug("\n✓ Embedding phase completed successfully")
    else:
        logger.debug(f"\n✗ Embedding phase failed: {embed_result['status']}")

    logger.info('Matching Service running on http://0.0.0.0:9020')
    app.run(host='0.0.0.0', port=9020)