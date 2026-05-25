import { areaPathFor, maxValue, pathFor, plot, pointsFor, SvgChart, valueOf, xAt, yAt } from './SvgChart';

const data = [
  { name: 'Q1', 收入: 800, 支出: 300, 利润: 500, 增长率: 20 },
  { name: 'Q2', 收入: 967, 支出: 467, 利润: 500, 增长率: 10 },
  { name: 'Q3', 收入: 1098, 支出: 749, 利润: 349, 增长率: 15 },
  { name: 'Q4', 收入: 1200, 支出: 880, 利润: 320, 增长率: 12 },
  { name: 'Q5', 收入: 1108, 支出: 600, 利润: 508, 增长率: 22 },
  { name: 'Q6', 收入: 1300, 支出: 700, 利润: 600, 增长率: 25 },
];

export function ComposedChartExample() {
  const amountMax = maxValue(data, ['收入', '支出', '利润']);
  const growthMax = maxValue(data, ['增长率']);
  const barWidth = Math.min(24, plot.innerWidth / data.length / 4);
  const profitPoints = pointsFor(data, '利润', amountMax);
  const growthPoints = data.map((point, index) => ({
    x: xAt(index, data.length),
    y: yAt(valueOf(point, '增长率'), growthMax),
  }));

  return (
    <SvgChart
      data={data}
      max={amountMax}
      legend={[
        { label: '收入', color: 'var(--color-accent)' },
        { label: '支出', color: '#82ca9d' },
        { label: '利润', color: '#8884d8' },
        { label: '增长率', color: '#ff7300' },
      ]}
    >
      <path d={areaPathFor(profitPoints)} fill="#8884d8" opacity="0.18" />
      <path d={pathFor(profitPoints)} fill="none" stroke="#8884d8" strokeWidth="2.5" />
      {data.map((point, index) => {
        const center = xAt(index, data.length);
        return (
          <g key={point.name}>
            {(['收入', '支出'] as const).map((key, keyIndex) => {
              const height = (valueOf(point, key) / amountMax) * plot.innerHeight;
              const x = center + (keyIndex === 0 ? -barWidth - 3 : 3);
              return <rect key={key} x={x} y={plot.bottom - height} width={barWidth} height={height} rx="5" fill={keyIndex === 0 ? 'var(--color-accent)' : '#82ca9d'} />;
            })}
          </g>
        );
      })}
      <path d={pathFor(growthPoints)} fill="none" stroke="#ff7300" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
      {growthPoints.map((point, index) => <circle key={index} cx={point.x} cy={point.y} r="4" fill="var(--color-bg-surface)" stroke="#ff7300" strokeWidth="2" />)}
      <text x={plot.right} y={plot.top - 8} textAnchor="end" className="fill-[var(--color-text-muted)] text-[11px]">
        右轴：增长率最高 {growthMax}%
      </text>
    </SvgChart>
  );
}
