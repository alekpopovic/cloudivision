from fastapi import FastAPI

from app.service import health_payload

app = FastAPI(title="cloudivision Python example")


@app.get("/healthz")
def health() -> dict[str, str]:
    return health_payload()
