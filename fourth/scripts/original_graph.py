from scripts.definitions import *
import networkx as nx

# Load the edge list and create a directed Graph
with open("../hamster.edgelist", 'rb') as fh:
    G = nx.read_edgelist(fh, create_using=nx.DiGraph())


##################
# Original Graph #
##################

base = pagerank(G)
visualize(base)
display_rank(base)
