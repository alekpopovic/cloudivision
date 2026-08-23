# Python FastAPI example

A small FastAPI service with business logic covered by the Python standard-library test runner.

```sh
cd examples/python-fastapi
python -m unittest discover -s tests
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --port 8080
```

```sh
docker build -t cloudivision-fastapi:local .
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f examples/python-fastapi/cloudivision.yaml
```

The unit test is offline; package and base-image downloads use public sources and no paid service.
