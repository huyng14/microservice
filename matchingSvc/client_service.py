import asyncio
import aiohttp

async def call_matching_service(timeout=60):
    """
    Async client to call the /matching endpoint from the matching service.
    
    Returns:
        dict: Response from the /matching endpoint
    """
    url = 'http://localhost:9020/matching'
    
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url, timeout=timeout) as response:
                if response.status == 200:
                    data = await response.json()
                    print(f"✓ Matching service response: {data}")
                    return data
                else:
                    print(f"✗ Error: Status code {response.status}")
                    return None
    except Exception as e:
        print(f"✗ Failed to call matching service: {str(e)}")
        return None

async def call_matching_multiple_times(times=3):
    """
    Call the /matching endpoint multiple times concurrently.
    
    Args:
        times (int): Number of times to call the endpoint
        
    Returns:
        list: List of responses from all requests
    """
    print(f"\nCalling /matching endpoint {times} times concurrently...")
    
    tasks = [call_matching_service() for _ in range(times)]
    results = await asyncio.gather(*tasks)
    
    return results

if __name__ == '__main__':
    print("Starting async client service...")
    
    # Call matching service once
    print("\n" + "="*60)
    print("Single call to /matching")
    print("="*60)
    result = asyncio.run(call_matching_service())
    
    # Call matching service multiple times concurrently
    print("\n" + "="*60)
    print("Multiple concurrent calls to /matching")
    print("="*60)
    results = asyncio.run(call_matching_multiple_times(times=3))
    print(f"\n✓ Received {len(results)} responses")
