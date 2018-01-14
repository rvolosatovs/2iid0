from scripts.definitions import *
import random
import networkx as nx

# Load the edge list and create a directed Graph
with open("../hamster.edgelist", 'rb') as fh:
    G = nx.read_edgelist(fh, create_using=nx.DiGraph())


#################
# Evolved Graph #
#################

# Create a copy of the original graph
Gx = G.copy()

# Randomly remove edges from the graph
# Gx.remove_edges_from(random.sample(G.edges(), 20*G.number_of_edges()//100))

# Randomly remove leaves from the graph
leaves = get_leaves(Gx)
leaf_edges = Gx.in_edges(leaves)

# Remove edges only
# Gx.remove_edges_from(random.sample(list(leaf_edges), 20*len(leaf_edges)//100))

# Remove both edges and nodes
Gx.remove_nodes_from(random.sample(leaves, 20*len(leaves)//100))

# Calculate the PageRank
pr = pagerank(Gx)

# Show some statistics about the new graph
visualize(pr)
display_rank(pr)
