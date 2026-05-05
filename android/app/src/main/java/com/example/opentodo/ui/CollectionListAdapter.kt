package com.example.opentodo.ui

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.example.opentodo.R

class CollectionListAdapter(
    private val collections: MutableList<UiCollection>,
    private val onClick: (UiCollection) -> Unit
) : RecyclerView.Adapter<CollectionListAdapter.CollectionViewHolder>() {

    fun setCollections(newCollections: List<UiCollection>) {
        collections.clear()
        collections.addAll(newCollections)
        notifyDataSetChanged()
    }

    class CollectionViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        val name: TextView = view.findViewById(R.id.collectionName)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): CollectionViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_collection, parent, false)
        return CollectionViewHolder(view)
    }

    override fun getItemCount(): Int = collections.size

    override fun onBindViewHolder(holder: CollectionViewHolder, position: Int) {
        val collection = collections[position]
        holder.name.text = collection.name
        holder.itemView.setOnClickListener { onClick(collection) }
    }
}
// TODO
