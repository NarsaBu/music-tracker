import type { Meta, StoryObj } from '@storybook/react-vite';
import { LedBar } from './LedBar';

const meta = { title: 'UI/LedBar', component: LedBar } satisfies Meta<typeof LedBar>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Running: Story = { args: { value: 37, label: 'Прогресс', active: true } };
export const Done: Story = { args: { value: 100, label: 'Прогресс' } };
export const Empty: Story = { args: { value: 0, label: 'Прогресс' } };
