from fastapi import FastAPI
from store import Store
import os

app = FastAPI()
cur_store = Store()

@app.get("/rl/{ip}/{timestamp}")
async def root(ip: str, timestamp: int):
	result = cur_store.add_entry(ip, timestamp)
	return result


if __name__ == "__main__":
	import uvicorn
	uvicorn.run(app, port=int(os.getenv("PORT", 8080)))
