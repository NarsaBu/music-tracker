import type { Meta, StoryObj } from '@storybook/react-vite';
import { VuMeter } from './VuMeter';

const meta = {
  title: 'UI/VuMeter',
  component: VuMeter,
  args: { label: 'Прогресс синхронизации' },
} satisfies Meta<typeof VuMeter>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Idle: Story = { args: { value: 0 } };
export const Halfway: Story = { args: { value: 50 } };
export const Peak: Story = { args: { value: 100 } };
