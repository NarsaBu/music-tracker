import type { Meta, StoryObj } from '@storybook/react-vite';
import { Led } from './Led';

const meta = { title: 'UI/Led', component: Led } satisfies Meta<typeof Led>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Off: Story = { args: { state: 'off', label: 'Выключено' } };
export const On: Story = { args: { state: 'on', label: 'Включено' } };
export const Ok: Story = { args: { state: 'ok', label: 'В сети' } };
export const Warn: Story = { args: { state: 'warn', label: 'Внимание' } };
export const Blinking: Story = { args: { state: 'on', blink: true, label: 'Синхронизация' } };
