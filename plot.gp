set terminal pdf
set output 'scc.pdf'

set style function linespoints
set style line 1 \
    linecolor rgb '#0060ad' \
    linetype 1 linewidth 2 \
    pointtype 7 pointsize .2

set style line 2 \
    linecolor rgb '#dd181f' \
    linetype 1 linewidth 2 \
    pointtype 5 pointsize .2

# define axis
set style line 11 lc rgb '#808080' lt 1

# y1
set ytics 5 nomirror tc ls 11
set ylabel 'average degree' tc ls 1

# y2
set y2tics nomirror tc ls 11 ('' 0,'5' 5,'10' 10,'15' 15,'20' 20)
set y2label 'largest component size' tc ls 2

set xtics ('1.0' '1.0','2.0' '2.0','3.0' '3.0','4.0' '4.0','5.0' '5,0'); 

set style line 12 lc rgb '#808080' lt 0 lw 1
set grid back ls 12

plot 'scc.dat' using 4:xticlabels(1) ls 1, \
     ''        using 7:xticlabels(1) ls 2 axes x1y2