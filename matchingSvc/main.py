import time
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
CORS(app)  # Enable CORS for all routes
# Configure logging with line numbers
logging.basicConfig(
    level=logging.DEBUG,
    format='[DEBUG] %(filename)s:%(lineno)d - %(message)s'
)
logging.disable(logging.CRITICAL)  # Disable all logging
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

def compare_phase(consultant_id=None, assignment_id=None, top_k=10, explain=True, database="project", 
                  collection_consultants="consultants", collection_assignments="jobs"):
    """
    Phase 3: Find best matches between consultants and assignments.
    
    Args:
        consultant_id: ObjectId of consultant to match with assignments
        assignment_id: ObjectId of assignment to match with consultants
        top_k: number of top matches to return
        explain: whether to generate explanations using Gemini
    
    Returns:
        dict with keys 'error' and 'response'
    """
    print("\n" + "="*60)
    print("PHASE 3: COMPARING & MATCHING")
    print("="*60)
    
    # Validate consultant_id
    if not consultant_id:
        error_msg = "Error: Please provide consultant_id"
        print(error_msg)
        return {"error": error_msg, "response": None}
    
    try:
        print(f"\nSearching for best assignments for consultant {consultant_id}...")
        assignments_sorted = compare_consultant_with_assignments(consultant_id, database, collection_consultants, collection_assignments)
        
        if not assignments_sorted:
            error_msg = "No matches found or no data available."
            print(error_msg)
            return {"error": error_msg, "response": None}
        
        consultant = find_ID_from_database(database, collection_consultants, consultant_id)
        if not consultant:
            error_msg = f"Consultant {consultant_id} not found in database."
            print(error_msg)
            return {"error": error_msg, "response": None}
        
        top_matches = assignments_sorted[:top_k]
        print(f"\n{'Rank':<6} {'Similarity':<12} {'Assignment ID':<26} {'Title':<40}")
        print("=" * 90)
        
        matches = []
        for rank, (_id, score) in enumerate(top_matches, 1):
            assignment = find_objectID_from_database(database, collection_assignments, _id)
            if not assignment:
                continue
            
            title = assignment.get("title", "N/A")[:37]
            print(f"{rank:<6} {score:<12.4f} {assignment.get('id', 'N/A'):<26} {title:<40}")
            
            explanation = None
            if explain:
                try:
                    explanation = generate_explanation_toConsultant(
                        consultant.get("Work Experience", ""),
                        assignment.get("Description", ""),
                        score
                    )
                except Exception as e:
                    explanation = f"(Could not generate explanation: {str(e)})"
            
            matches.append({
                "rank": rank,
                "jobId": assignment.get('id', 'N/A'),
                "jobTitle": title,
                "score": score,
                "company": assignment.get("company", "N/A"),
                "location": assignment.get("location", "N/A"),
                "explanation": explanation
            })
            
            if explanation:
                print(f"\t{explanation}\n")
        
        response = {
            "consultant_id": str(consultant_id),
            "matches": matches
        }
        return {"error": None, "response": response}
    
    except Exception as e:
        error_msg = f"Exception in compare_phase: {str(e)}"
        logger.debug(error_msg)
        return {"error": error_msg, "response": None}
    
def get_matched_score(consultant_id=None, assignment_id=None, explain=False, database="project", 
                  collection_consultants="consultants", collection_assignments="jobs"):
    if not consultant_id or not assignment_id:
        error_msg = "Error: Please provide both consultant_id and assignment_id"
        print(error_msg)
        return {"error": error_msg, "response": None}
    try:
        print(f"\nCalculating similarity score between consultant {consultant_id} and assignment {assignment_id}...")
        consultant = find_ID_from_database(database, collection_consultants, consultant_id)
        assignment = find_ID_from_database(database, collection_assignments, assignment_id)
        
        if not consultant:
            error_msg = f"Consultant {consultant_id} not found in database."
            print(error_msg)
            return {"error": error_msg, "response": None}
        
        if not assignment:
            error_msg = f"Assignment {assignment_id} not found in database."
            print(error_msg)
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
        logger.debug(error_msg)
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

if __name__ == '__main__':
    embed_result = embed_phase("project", "consultants", "jobs")
    if embed_result["status"] == "completed":
        logger.debug("\n✓ Embedding phase completed successfully")
    else:
        logger.debug(f"\n✗ Embedding phase failed: {embed_result['status']}")

    print('Matching Service running on http://0.0.0.0:9020')
    app.run(host='0.0.0.0', port=9020)