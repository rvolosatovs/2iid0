import networkx as nx

# load the edge list and create a directed Graph
fh = open("hamster.edgelist", 'rb')
G = nx.read_edgelist(fh, create_using=nx.DiGraph())
fh.close()

# calculate PageRank directly use
# the function from NetworkX
# more details about this function
# can be found 
# https://networkx.github.io/documentation/latest/reference/algorithms/generated/networkx.algorithms.link_analysis.pagerank_alg.pagerank.html
pr = nx.pagerank(G, alpha=0.85)
# the return value pr is dict in Python
# in which the keys are the node indices
# and the values are corresponding PagRank values
print(pr)
