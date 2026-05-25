import { maxValue, pathFor, pointsFor, SvgChart } from './SvgChart';

const data = [
  { name: '周一', 访问量: 4000, 用户数: 2400 },
  { name: '周二', 访问量: 3000, 用户数: 1398 },
  { name: '周三', 访问量: 2000, 用户数: 9800 },
  { name: '周四', 访问量: 2780, 用户数: 3908 },
  { name: '周五', 访问量: 1890, 用户数: 4800 },
  { name: '周六', 访问量: 2390, 用户数: 3800 },
  { name: '周日', 访问量: 3490, 用户数: 4300 },
];

export function LineChartExample() {
  const series = [
    { key: '访问量', color: 'var(--color-accent)' },
    { key: '用户数', color: '#82ca9d' },
  ];
  const max = maxValue(data, series.map((item) => item.key));

  return (
    <SvgChart data={data} max={max} legend={series.map(({ key, color }) => ({ label: key, color }))}>
      {series.map(({ key, color }) => {
        const points = pointsFor(data, key, max);
        return (
          <g key={key}>
            <path d={pathFor(points)} fill="none" stroke={color} strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
            {points.map((point, index) => <circle key={index} cx={point.x} cy={point.y} r="4" fill="var(--color-bg-surface)" stroke={color} strokeWidth="2" />)}
          </g>
        );
      })}
    </SvgChart>
  );
}
