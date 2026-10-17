import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { RotarySelector, type RotarySelectorProps } from './RotarySelector';

function Stateful(props: RotarySelectorProps<string>) {
  const [value, setValue] = useState(props.value);
  return <RotarySelector {...props} value={value} onChange={setValue} />;
}

const meta = {
  title: 'UI/RotarySelector',
  component: RotarySelector,
  render: (args) => <Stateful {...(args as RotarySelectorProps<string>)} />,
} satisfies Meta<typeof RotarySelector>;
export default meta;
type Story = StoryObj<typeof meta>;

export const TwoPositions: Story = {
  args: {
    label: 'Частота',
    value: 'daily',
    onChange: () => {},
    options: [
      { value: 'daily', label: 'Daily' },
      { value: 'weekly', label: 'Weekly' },
    ],
  },
};

export const SevenPositions: Story = {
  args: {
    label: 'День недели',
    value: '5',
    onChange: () => {},
    options: ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'].map((label, i) => ({
      value: String((i + 1) % 7),
      label,
    })),
  },
};
