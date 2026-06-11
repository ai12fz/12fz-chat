"""
12FZ AI Service setup.
"""
from setuptools import setup, find_packages

setup(
    name="12fz-ai-service",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        "fastapi>=0.110.0",
        "uvicorn[standard]>=0.27.0",
        "httpx>=0.26.0",
        "pydantic>=2.0.0",
    ],
)
