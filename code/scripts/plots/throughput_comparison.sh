python3 bar_chart_plot.py \
  -d throughput_comparison.tsv \
  -o throughput_comparison.pdf \
  --ylabel "{Throughput (kRPS)}" \
  --xlabel "Number of Goroutines" \
  --ymin 3000 --ymax 50000 \
  --xscale 0.03 --yscale 0.0175 \
  --colors "green,blue,violet,black,cyan,red" \
  --patterns "grid,crosshatch,dots,north east lines,crosshatch dots,north west lines"