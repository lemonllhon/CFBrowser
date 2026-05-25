import { maxValue, plot, SvgChart, valueOf, xAt } from './SvgChart';

const data = [
  { name: '1月', 产品A: 4000, 产品B: 2400 },
  { name: '2月', 产品A: 3000, 产品B: 1398 },
  { name: '3月', 产品A: 2000, 产品B: 9800 },
  { name: '4月', 产品A: 2780, 产品B: 3908 },
  { name: '5月', 产品A: 1890, 产品B: 4800 },
  { name: '6月', 产品A: 2390, 产品B: 3800 },
  { name: '7月', 产品A: 3490, 产品B: 4300 },
];

export function BarChartExample() {
  const keys = ['产品A', '产品B'];
  const colors = ['var(--color-accent)', '#82ca9d'];
  const max = maxValue(data, keys);
  const groupWidth = Math.min(58, plot.innerWidth / data.length - 18);
  const barWidth = groupWidth / keys.length - 4;

  return (
    <SvgChart data={data} max={max} legend={keys.map((label, index) => ({ label, color: colors[index] }))}>
      {data.map((point, index) => {
        const center = xAt(index, data.length);
        return keys.map((key, keyIndex) => {
          const height = (valueOf(point, key) / max) * plot.innerHeight;
          const x = center - groupWidth / 2 + keyIndex * (barWidth + 8);
          const y = plot.bottom - height;
          return <rect key={`${point.name}-${key}`} x={x} y={y} width={barWidth} height={height} rx="5" fill={colors[keyIndex]} />;
        });
      })}
    </SvgChart>
  );
}
