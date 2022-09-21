python3 bar_chart_plot.py \
  -d write_latency.tsv \
  -o write_latency.pdf \
  --ylabel "{Write Latency (us)}" \
  --xlabel "Number of Goroutines" \
  --ymin 900 --ymax 30000 \
  --xscale 0.03 --yscale 0.0175 \
  --colors "green,blue,violet,black,cyan,red" \
  --patterns "grid,crosshatch,dots,north east lines,crosshatch dots,north west lines"