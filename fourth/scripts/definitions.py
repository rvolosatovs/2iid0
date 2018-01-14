import numpy as np
import matplotlib.pyplot as plt
import pandas as pd
import networkx as nx


# Calculate the PageRank for the nodes in directed graph G
def pagerank(G):
    return pd.DataFrame.from_dict(nx.pagerank(G), orient='index').rename(columns={0: 'pagerank'})


# Visualize a pandas DataFrame
def visualize(df):
    print('Pagerank distribution')
    df['pagerank'].plot.hist(bins=100)
    plt.title('PageRank Distribution')
    plt.show()

    print('Log scale')
    plt.title('PageRank Distribution (log scale)')
    df['pagerank'].apply(np.log).plot.hist(bins=100)
    plt.show()


# Sort a pandas DataFrame by PageRank
def sort_by_pagerank(df):
    return df.sort_values(by='pagerank', ascending=False)


# Display some PageRank statistics from a pandas DataFrame
def display_rank(df):
    # Sort the DataFrame by PageRank
    ranking = sort_by_pagerank(df)

    print("Top 10:\n")
    print(ranking.head(10))

    print("Lowest 10:\n")
    print(ranking.tail(10))


def compute_error(base, df):
    br = sort_by_pagerank(base)
    nr = sort_by_pagerank(df)


def get_nodes_by_degree(graph, in_degree, out_degree):
    # Initialize a list to store the nodes in
    nodes = []

    # Loop through all the nodes and the degrees
    for node in graph.nodes():
        if graph.in_degree(node) == in_degree and graph.out_degree(node) == out_degree:
            nodes.append(node)

    return nodes


def get_leaves(graph):
    return get_nodes_by_degree(graph, 1, 0)
