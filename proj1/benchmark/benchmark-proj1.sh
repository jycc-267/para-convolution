#!/bin/bash
#
#SBATCH --mail-user=jycchien@uchicago.edu
#SBATCH --mail-type=ALL
#SBATCH --job-name=proj1_benchmark 
#SBATCH --output=./slurm/out/%j.%N.stdout
#SBATCH --error=./slurm/out/%j.%N.stderr
#SBATCH --chdir=/home/jycchien/parallel/project-1-jycc-267/proj1/benchmark
#SBATCH --partition=debug 
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=16
#SBATCH --mem-per-cpu=900
#SBATCH --exclusive
#SBATCH --time=3:00:00


module load golang/1.19

mkdir -p results

# Run sequential baseline 5 times for each dataset
for dataset in small mixture big; do
    for run in {1..5}; do
        /usr/bin/time -f "%e" go run ../editor/editor.go $dataset 2>> results/${dataset}_sequential.txt
    done
done

# Run parallel versions with different thread counts
for mode in parfiles parslices; do
    for threads in 2 4 6 8 12; do
        for dataset in small mixture big; do
            for run in {1..5}; do
                /usr/bin/time -f "%e" go run ../editor/editor.go $dataset $mode $threads 2>> results/${dataset}_${mode}_${threads}.txt
            done
        done
    done
done

# Generate plots using Python
python3 plot.py