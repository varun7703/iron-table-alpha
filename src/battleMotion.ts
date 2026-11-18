export type MotionUnit={id:string;owner:string;type:string;territory:number;count:number;health:number;ration:number;fed:boolean;motionX?:number;motionY?:number};
export type MotionEvent={text:string;from:number;to:number;kind:string};
export type MotionFrame={label:string;units:MotionUnit[];owners:string[];events:MotionEvent[]};
export const clamp=(x:number)=>Math.max(0,Math.min(1,x));
export function sampleBattle(before:MotionFrame,after:MotionFrame,t:number,tiles:{x:number;y:number}[],reduced:boolean){
 const impact=.62,move=clamp((t-.08)/.54),travel=reduced?(t<impact?0:1):move*move*(3-2*move);
 const units=before.units.flatMap(old=>{
  const next=after.units.find(u=>u.id===old.id);
  if(!next)return t<impact?[old]:[];
  const a=tiles[old.territory],b=tiles[next.territory];
  const healthT=reduced?(t<impact?0:1):clamp((t-impact)/.2);
  return [{...next,count:t<impact?old.count:next.count,health:old.health+(next.health-old.health)*healthT,territory:t<impact?old.territory:next.territory,motionX:a.x+(b.x-a.x)*travel,motionY:a.y+(b.y-a.y)*travel-Math.sin(travel*Math.PI)*(old.territory!==next.territory?16:0)}];
 });
 for(const u of after.units){if(!before.units.some(v=>v.id===u.id))units.push(u)}
 return {units,owners:t<impact?before.owners:after.owners};
}
