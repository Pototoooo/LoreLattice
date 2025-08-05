#!/usr/bin/env python
# -*- coding: utf-8 -*-

import sys
import os
import logging
from paddleocr import PaddleOCR

# 添加当前目录到Python路径
current_dir = os.path.dirname(os.path.abspath(__file__))
if current_dir not in sys.path:
    sys.path.append(current_dir)

# 导入ImageParser
from parser.image_parser import ImageParser

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


def init_ocr_model():
    PaddleOCR()
