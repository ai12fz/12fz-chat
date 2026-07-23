<template>
  <div v-if="visible" class="dialog-overlay" @click.self="close">
    <div class="dialog-card">
      <h3>添加好友</h3>
      <label style="display:block;margin-bottom:4px;font-size:13px;color:#666">请输入好友的用户ID</label>
      <input v-model="friendId" placeholder="输入 go.12fz.com 用户ID" @keyup.enter="submit" />
      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="tip" class="tip">{{ tip }}</div>
      <div class="dialog-btns">
        <button @click="close" class="btn-cancel">取消</button>
        <button @click="submit" :disabled="!friendId.trim()" class="btn-ok">添加</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { addFriend } from "../api";

const props = defineProps<{ visible: boolean }>();
const emit = defineEmits(["close", "added"]);
const friendId = ref("");
const error = ref("");
const tip = ref("");

async function submit() {
  error.value = "";
  tip.value = "";
  const fid = friendId.value.trim();
  if (!fid) return;
  const token = localStorage.getItem("token") || "";
  const myId = token.startsWith("session-") ? token.slice(8) : token;
  try {
    await addFriend(myId, fid);
    emit("added", fid);
    friendId.value = "";
    tip.value = "请求已发送，等待对方同意";
    setTimeout(() => { tip.value = ""; close(); }, 1500);
  } catch (e: any) {
    error.value = e.response?.data?.error || "添加失败";
  }
}

function close() {
  error.value = "";
  friendId.value = "";
  tip.value = "";
  emit("close");
}
</script>

<style scoped>
.dialog-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,.4);
  display: flex; align-items: center; justify-content: center; z-index: 200;
}
.dialog-card {
  background: #fff; border-radius: 8px; padding: 24px;
  width: 380px; box-shadow: 0 4px 12px rgba(0,0,0,.15);
}
.dialog-card h3 { margin: 0 0 16px; font-size: 18px; }
.dialog-card input {
  width: 100%; padding: 8px 12px; border: 1px solid #d9d9d9;
  border-radius: 4px; font-size: 14px; box-sizing: border-box;
}
.error { color: #f5222d; font-size: 13px; margin: 8px 0; }
.tip { color: #52c41a; font-size: 13px; margin: 8px 0; }
.dialog-btns {
  display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px;
}
.btn-cancel {
  padding: 6px 16px; border: 1px solid #d9d9d9;
  border-radius: 4px; background: #fff; cursor: pointer;
}
.btn-ok {
  padding: 6px 16px; border: none; border-radius: 4px;
  background: #1890ff; color: #fff; cursor: pointer;
}
.btn-ok:disabled { opacity: .5; cursor: not-allowed; }
</style>
