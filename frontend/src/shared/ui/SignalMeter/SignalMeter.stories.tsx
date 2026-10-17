import type { Meta, StoryObj } from '@storybook/react-vite';
import { SignalMeter } from './SignalMeter';

const meta = {
  title: 'UI/SignalMeter',
  component: SignalMeter,
  args: { label: 'Совпадение' },
} satisfies Meta<typeof SignalMeter>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Strong: Story = { args: { value: 100 } };
export const Medium: Story = { args: { value: 62 } };
export const Weak: Story = { args: { value: 18 } };
