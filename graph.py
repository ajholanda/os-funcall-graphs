##
# This file contains code to generate a graph from the 
# function calls from a specified set of source codes.
#
# Our first attempt is to generate the function calls graph 
# from Linux source code. To accomplish this task, we have 
# to download the Linux tarball, uncompress it, 
# parse the source files and write the graph in 
# a file to be used in other steps.
##
from asyncio.subprocess import DEVNULL
import logging
import os
import posixpath
import re
import subprocess
import sys
import tarfile
import urllib.parse
import urllib.request

import networkx as nx
from tqdm import tqdm

NAME = 'linux'      # name of the OS project
CEXT = '.tar.gz'    # tarball file extension
EXT = '.adj'
logging.basicConfig(level=logging.INFO)

def srcpath(version: str) -> str:
    """Return the relative path to source code directory.
    """
    return NAME + '-' + version

def download(version: str) -> str:
    MIRROR = 'https://mirrors.edge.kernel.org/pub/linux/kernel/'
    major = version.split('.')[0]
    dir = 'v' + major + '.x'
    fname = srcpath(version) + CEXT
    path = posixpath.join(dir, fname)
    url = urllib.parse.urljoin(MIRROR, path)
    dest = os.path.join('/tmp', fname)
    if not os.path.exists(dest):
        logging.info('downloading'.format(url))
        urllib.request.urlretrieve(url, dest)

    return dest

def uncompress(path: str):
    logging.info('uncompressing files to'.format(path))            
    file = tarfile.open(path)
    file.extractall('./')
    file.close()

def classify(fname, line):
    caller_m = re.search(r"^(\w+)\(\)", line)
    callee_m = re.search(r"^[ \t]+(\w+)\(\).*", line)

    if caller_m:
        return caller_m.group(1), 0
    elif callee_m:
        return callee_m.group(1), 1
    else:
        logging.warning('unknown pattern {}.{}'.format(fname, line))
        return None, 2

def add_edge(graph, src, dst):
    # Add the arc to graph with its weight updated
    if src not in graph:
            graph.add_node(src)

    if dst not in graph:
        graph.add_node(dst)

    if dst not in graph[src]:
        graph.add_edge(src, dst, weight=1)
    else: # increment weight
        graph[src][dst]['weight'] += 1

def parse(path: str) -> nx.Graph:
    graph = nx.Graph()
    flow_ok = False # state var to trace function call flow 

    ret = subprocess.run(['find', path, '-name', '*.c'], stdout=subprocess.PIPE)
    cpaths = ret.stdout.decode('utf8')
    logging.info('processing C files from {}/*...'.format(path))
    pbar = tqdm(cpaths.splitlines())
    for cpath in pbar:
        cpath = os.path.join('./', cpath)
        ret = subprocess.run(['cflow', '--depth', '2', cpath],
                             stdout=subprocess.PIPE, 
                             stderr=DEVNULL)
        if not ret.stdout:
            continue
        clines = ret.stdout.decode('utf8')
        for cline in clines.splitlines():
            func, cls = classify(cpath, cline)
            if cls == 0: # function definition = caller
                flow_ok = True # normalize flow
                caller = func
            elif cls == 1:
                if flow_ok:
                    callee = func
                    add_edge(graph, caller, callee)
            else: # unknown pattern
                # Wait next function call to normalize flow
                flow_ok = False
    return graph

def create(version: str) -> nx.Graph:
    path = srcpath(version)

    if not os.path.exists(path):
        tarpath = download(version)
        uncompress(tarpath)
    
    return parse(path)


def write(version: str):
    outfn = srcpath(version) + EXT 
    if os.path.exists(outfn):
        logging.warning('{} already exists'.format(outfn))
        return

    graph = create(version)
    nx.write_adjlist(graph, outfn)
    logging.info('wrote {}, a {} '.format(outfn, graph))

def read(fname: str) -> nx.Graph:
    logging.info('reading {}...'.format(fname))
    return nx.read_adjlist(fname)

if __name__ == '__main__':
    version = sys.argv[1] 
    write(version)

