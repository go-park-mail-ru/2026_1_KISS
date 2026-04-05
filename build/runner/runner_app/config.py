import logging
import os


logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"))

DEFAULT_TIMEOUT = float(os.getenv("EXEC_TIMEOUT_SECONDS", "15"))
MAX_TIMEOUT = float(os.getenv("MAX_EXEC_TIMEOUT_SECONDS", "120"))
KERNEL_NAME = os.getenv("KERNEL_NAME", "python3")
