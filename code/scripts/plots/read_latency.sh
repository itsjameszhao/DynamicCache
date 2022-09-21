python3 bar_chart_plot.py \
  -d read_latency.tsv \
  -o read_latency.pdf \
  --ylabel "{Read Latency (us)}" \
  --xlabel "Number of Goroutines" \
  --ymin 40 --ymax 1600 \
  --xscale 0.03 --yscale 0.0175 \
  --colors "green,blue,violet,black,cyan,red" \
  --patterns "grid,crosshatch,dots,north east lines,crosshatch dots,north west lines"