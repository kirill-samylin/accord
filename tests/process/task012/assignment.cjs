'use strict';
const {assert}=require('./harness.cjs');
async function run({native,create},pass){
 const settings={name:'QA assignment revision',description:'Synthetic QA',role:'developer',instructions:'QA no worker',provider:'custom',model:'qa-only'};
 const profile=native.call('profile_create',settings),task=create('QA assigned unavailable edited profile unique');
 native.call('assign',{task_key:task.key,version:task.version,mode:'profile',profile_id:profile.id,profile_version:1,reason:'QA assign'});
 assert(native.call('team').unclaimed.some(t=>t.key===task.key));
 native.call('profile_update',{...settings,profile_id:profile.id,version:1,instructions:'QA revised instructions'});
 assert(!native.call('team').unclaimed.some(t=>t.key===task.key));
 const latest=native.call('get',{task_key:task.key});native.call('assign',{task_key:task.key,version:latest.version,mode:'profile',profile_id:profile.id,profile_version:2,reason:'QA repin'});
 assert(native.call('team').unclaimed.some(t=>t.key===task.key));native.call('profile_archive',{profile_id:profile.id,version:2,state:'archived'});
 assert(!native.call('team').unclaimed.some(t=>t.key===task.key));pass('team-unclaimed-excludes-edited-and-archived-pinned-profiles');
}
module.exports={run};
