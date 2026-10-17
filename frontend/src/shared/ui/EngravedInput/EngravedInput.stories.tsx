import type { Meta, StoryObj } from '@storybook/react-vite';
import { EngravedInput } from './EngravedInput';

const meta = { title: 'UI/EngravedInput', component: EngravedInput } satisfies Meta<
  typeof EngravedInput
>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = { args: { label: 'Email', placeholder: 'me@example.com' } };
export const WithHint: Story = {
  args: { label: 'Время', type: 'time', defaultValue: '09:00', hint: 'В вашем часовом поясе' },
};
export const WithError: Story = {
  args: { label: 'Email', defaultValue: 'nope', error: 'Некорректный адрес' },
};
