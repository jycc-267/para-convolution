import numpy as np
import matplotlib.pyplot as plt

def get_min_runtime(filename):
    with open(filename) as f:
        times = [float(line.strip()) for line in f]
    return min(times)

thread_counts = [2, 4, 6, 8, 12]
datasets = ['small', 'mixture', 'big']
modes = ['bsp', 'bspsteal']

for mode in modes:
    plt.figure(figsize=(10, 6))
    
    for dataset in datasets:
        speedups = []
        sequential_time = get_min_runtime(f'results/{dataset}_sequential.txt')
        
        for threads in thread_counts:
            parallel_time = get_min_runtime(f'results/{dataset}_{mode}_{threads}.txt')
            # sequential time is the same as parallel versions with one thread
            speedup = sequential_time / parallel_time
            speedups.append(speedup)
            
        plt.plot(thread_counts, speedups, marker = 'o', label = dataset)
    
    plt.xlabel('Number of Threads')
    plt.ylabel('Speedup')
    plt.title(f'Editor Speedup Graph ({mode.upper()})')
    plt.grid(True)
    plt.legend()
    plt.savefig(f'speedup-{mode}.png')
    plt.close()
