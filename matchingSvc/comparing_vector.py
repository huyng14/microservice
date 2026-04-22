import numpy as np

from pymongo import MongoClient
from bson.binary import Binary
from bson.objectid import ObjectId

from sklearn.metrics.pairwise import cosine_similarity
from google import genai

client = genai.Client(api_key='Change to your API key')
mongo_uri = "change to your MongoDB URI"

def find_objectID_from_database(db_name, collection_name, object_id):
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]

        # Find one document by _id
        document = collection.find_one({"_id": object_id})

        return document

    except Exception as e:
        print(f"Error reading from MongoDB: {e}")
        return None
    
def find_ID_from_database(db_name, collection_name, id):
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]

        # Find one document by _id
        document = collection.find_one({"id": id})

        return document

    except Exception as e:
        print(f"Error reading from MongoDB: {e}")
        return None
    
def read_from_database(db_name, collection_name):
    try:
        client_mongo = MongoClient(mongo_uri)
        db = client_mongo[db_name]
        collection = db[collection_name]

        # Read all documents into a list
        documents = list(collection.find())

        return documents
    
    except Exception as e:
        print(f"Error reading from MongoDB: {e}")
        return None
    
def compare_1consultant_with_1assignment_and_explain(consultant_id, assignment_id, explain=False, database="project",                                         
                                                     collection_consultants="consultants_wEmbedding", 
                                                    collection_assignments="assignments_wEmbedding"):
    consultant = find_ID_from_database(database, collection_consultants, consultant_id)
    assignment = find_ID_from_database(database, collection_assignments, assignment_id)

    if not consultant or not assignment:
        print("Consultant or assignment not found.")
        return None

    # Perform comparison and explanation generation
    consultant_binary_data = consultant["experience_embedding"]
    # Convert back to numpy array
    work_experience_embedding = np.frombuffer(consultant_binary_data, dtype=np.float32)

    assignment_binary_data = assignment.get("description_embedding")
    # Convert back to numpy array
    description_embedding = np.frombuffer(assignment_binary_data, dtype=np.float32)

    if work_experience_embedding.size != description_embedding.size:
        print(f"Skipping assignment {assignment.get('_id', '<no id>')}: embedding size mismatch ({work_experience_embedding.size} vs {description_embedding.size})")
        return None
    
    similarity_matrix = cosine_similarity(
                work_experience_embedding.reshape(1, -1),
                description_embedding.reshape(1, -1)
            )
    score = float(similarity_matrix[0, 0])
    explanation = None
    if explain:
        try:
            explanation = generate_explanation_toHiring_manager(
                consultant.get("experience", ""),
                assignment.get("description", ""),
                score
            )
        except Exception as e:
            explanation = f"(Could not generate explanation: {str(e)})"
    response = {
        "consultant_id": str(consultant_id),
        "assignment_id": str(assignment_id),
        "score": score,
        "explanation": explanation
    }
    return response


def compare_consultant_with_assignments(consultant_id, database="project", 
                                        collection_consultants="consultants_wEmbedding", 
                                        collection_assignments="assignments_wEmbedding"):
    consultant = find_ID_from_database(database, collection_consultants, consultant_id)

    if consultant:
        binary_data = consultant["experience_embedding"]

        # Convert back to numpy array
        work_experience_embedding = np.frombuffer(binary_data, dtype=np.float32)
    else:
        print("No data found or error occurred.")
    
    assignments = read_from_database(database, collection_assignments)
    if not assignments:
        print("No assignments found or error occurred.")
        return
    
    results = []
    # Print all documents
    for doc in assignments:
        try:
            binary_data = doc.get("description_embedding")
            if binary_data is None:
                print(f"Skipping assignment {doc.get('_id', '<no id>')}: no embedding")
                continue

            # Convert back to numpy array
            embedding_back = np.frombuffer(binary_data, dtype=np.float32)

            if work_experience_embedding.size != embedding_back.size:
                print(f"Skipping assignment {doc.get('_id', '<no id>')}: embedding size mismatch ({work_experience_embedding.size} vs {embedding_back.size})")
                continue

            similarity_matrix = cosine_similarity(
                work_experience_embedding.reshape(1, -1),
                embedding_back.reshape(1, -1)
            )
            score = float(similarity_matrix[0, 0])
            results.append((doc.get("_id"), score))
        except Exception as e:
            print(f"Error computing similarity for assignment {doc.get('_id', '<no id>')}: {e}")
    # Sort by similarity descending
    results_sorted = sorted(results, key=lambda x: x[1], reverse=True) 
    return results_sorted

def compare_assignment_with_consultants(assignment_id, database="project", 
                                        collection_consultants="consultants_wEmbedding", 
                                        collection_assignments="assignments_wEmbedding"):
    assignment = find_ID_from_database(database, collection_assignments, assignment_id)

    if assignment:
        binary_data = assignment["description_embedding"]

        # Convert back to numpy array
        description_embedding = np.frombuffer(binary_data, dtype=np.float32)
    else:
        print("No data found or error occurred.")
    
    consultants = read_from_database(database, collection_consultants)
    if not consultants:
        print("No consultants found or error occurred.")
        return []

    results = []
    for doc in consultants:
        try:
            binary_data = doc.get("work_experience_embedding")
            if binary_data is None:
                print(f"Skipping consultant {doc.get('_id', '<no id>')}: no embedding")
                continue

            # Convert back to numpy array
            embedding_back = np.frombuffer(binary_data, dtype=np.float32)

            if description_embedding.size != embedding_back.size:
                print(f"Skipping consultant {doc.get('_id', '<no id>')}: embedding size mismatch ({description_embedding.size} vs {embedding_back.size})")
                continue

            similarity_matrix = cosine_similarity(
                description_embedding.reshape(1, -1),
                embedding_back.reshape(1, -1)
            )
            score = float(similarity_matrix[0, 0])
            results.append((doc.get("_id"), score))
        except Exception as e:
            print(f"Error computing similarity for consultant {doc.get('_id', '<no id>')}: {e}")

    # Sort by similarity descending
    results_sorted = sorted(results, key=lambda x: x[1], reverse=True)

    return results_sorted

def interpret_similarity(score):
    if score >= 0.70:
        return "very strong alignment in role and expertise"
    elif score >= 0.60:
        return "strong overlap in skillset and background"
    elif score >= 0.45:
        return "some related experience but different specialization focus"
    elif score >= 0.25:
        return "mostly different backgrounds with limited overlap"
    else:
        return "little or no meaningful similarity"
    
def generate_explanation_toHiring_manager(profile, assignment, score):
    meaning = interpret_similarity(score)
    # print("Generating explanation with meaning:", meaning, profile, assignment)

    prompt = f"""
        Prompt with Profile and Assignment, and the cosine similarity result
        Compare the following a consultant profile and Assignment description, describe their similarity as if explaining to a hiring manager.
        Do not mention cosine similarity or math. Focus on meaningful overlap in skills, role experience, and domain background.
        Similarity interpretation: {meaning}

        Profile:
        {profile}
        Assignment description:
        {assignment}
        Write 3-5 sentences, professional but natural.
    """
    response = client.models.generate_content(
        model="gemini-2.5-flash",
        contents=prompt
    )
    return response.text.strip()

def generate_explanation_toConsultant(profile, assignment, score):
    meaning = interpret_similarity(score)
    # print("Generating explanation with meaning:", meaning, profile, assignment)

    prompt = f"""
        Prompt with Profile and Assignment, and the cosine similarity result
        Compare the following a consultant profile and Assignment description, describe their similarity as if explaining to a consultant/ candidate.
        Do not mention cosine similarity or math. Focus on meaningful overlap in skills, role experience, and domain background.
        Similarity interpretation: {meaning}

        Profile:
        {profile}
        Assignment description:
        {assignment}
        Write 3-5 sentences, professional but natural.
    """
    response = client.models.generate_content(
        model="gemini-2.5-flash",
        contents=prompt
    )
    return response.text.strip()