#!/usr/bin/env python3
"""
LoreLattice MCP Server Package

A Model Context Protocol server that provides access to the LoreLattice knowledge management API.
"""

__version__ = "1.0.0"
__author__ = "Pototoooo"
__description__ = "LoreLattice MCP Server - Model Context Protocol server for LoreLattice API"

from .lorelattice_mcp_server import LoreLatticeClient, run

__all__ = ["LoreLatticeClient", "run"]
