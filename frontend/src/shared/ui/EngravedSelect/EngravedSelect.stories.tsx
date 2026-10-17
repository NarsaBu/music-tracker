import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { EngravedSelect, type EngravedSelectProps } from './EngravedSelect';

const meta = {
  title: 'UI/EngravedSelect',
  component: EngravedSelect,
} satisfies Meta<typeof EngravedSelect>;

export default meta;
type Story = StoryObj<typeof meta>;

const options = [
  { value: '', label: 'Все' },
  { value: 'synced', label: 'Проверены' },
  { value: 'pending', label: 'Ждут проверки' },
];

function Controlled(args: EngravedSelectProps) {
  const [value, setValue] = useState(args.value);
  return <EngravedSelect {...args} value={value} onChange={setValue} />;
}

export const Default: Story = {
  args: { label: 'Статус', options, value: '', onChange: () => {} },
  render: (args) => <Controlled {...args} />,
};
