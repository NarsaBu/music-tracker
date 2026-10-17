import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { ToggleSwitch, type ToggleSwitchProps } from './ToggleSwitch';

function Stateful(props: Omit<ToggleSwitchProps, 'onChange'>) {
  const [checked, setChecked] = useState(props.checked);
  return <ToggleSwitch {...props} checked={checked} onChange={setChecked} />;
}

const meta = {
  title: 'UI/ToggleSwitch',
  component: ToggleSwitch,
  render: (args) => <Stateful {...args} />,
} satisfies Meta<typeof ToggleSwitch>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Off: Story = { args: { label: 'Album', checked: false, onChange: () => {} } };
export const On: Story = { args: { label: 'Album', checked: true, onChange: () => {} } };
export const WithDescription: Story = {
  args: {
    label: 'Featuring',
    description: 'Релизы, где исполнитель — приглашённый участник',
    checked: false,
    onChange: () => {},
  },
};
export const Disabled: Story = {
  args: { label: 'Album', checked: true, disabled: true, onChange: () => {} },
};
